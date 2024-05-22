package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type ProducerClientParams struct {
	Conn *nats.Conn
}

type ProducerClient struct {
	Conn *nats.Conn
	mu   sync.RWMutex // Protects access to activeStreamIds and activeConnHeartBeatRequestSubjects

	activeStreamIds                    map[string][]string // A map of ConnId to StreamId that are active
	activeConnHeartBeatRequestSubjects map[string]string   // A map of ConnId to the subject where a heart beat request should be sent
	heartBeatCancelFunc                context.CancelFunc
}

func NewProducerClient(params ProducerClientParams) (*ProducerClient, error) {
	nc := &ProducerClient{
		Conn:                               params.Conn,
		activeStreamIds:                    make(map[string][]string),
		activeConnHeartBeatRequestSubjects: make(map[string]string),
	}

	return nc, nil
}

func (pc *ProducerClient) AddConnDetails(ctx context.Context, connDetails *ConnectionDetails) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.activeStreamIds[connDetails.ConnId] = append(pc.activeStreamIds[connDetails.ConnId], connDetails.StreamId)
	pc.activeConnHeartBeatRequestSubjects[connDetails.ConnId] = connDetails.HeartBeatRequestSub

	if pc.heartBeatCancelFunc == nil {
		go pc.heartBeat(ctx)
	}

}

func (pc *ProducerClient) RemoveConnDetails(connDetails *ConnectionDetails) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	activeStreamIdsForConn, ok := pc.activeStreamIds[connDetails.ConnId]
	if !ok {
		return
	}

	for i, v := range activeStreamIdsForConn {
		if v == connDetails.StreamId {
			activeStreamIdsForConn = append(activeStreamIdsForConn[:i], activeStreamIdsForConn[i+1:]...)
		}
	}

	if len(activeStreamIdsForConn) == 0 {
		delete(pc.activeStreamIds, connDetails.ConnId)
		delete(pc.activeConnHeartBeatRequestSubjects, connDetails.StreamId)
	}

	if len(pc.activeStreamIds) == 0 && pc.heartBeatCancelFunc != nil {
		pc.heartBeatCancelFunc()
		pc.heartBeatCancelFunc = nil
	}

}

func (pc *ProducerClient) heartBeat(ctx context.Context) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	pc.heartBeatCancelFunc = cancel

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-ctxWithCancel.Done():
			return

		case <-ticker.C:

			results := make(map[string][]string)

			for c, v := range pc.activeConnHeartBeatRequestSubjects {
				msg, err := pc.Conn.Request(v, nil, 5*time.Second)
				if err != nil {
					results[c] = []string{}
					continue
				}

				var heartBeatResponse HeartBeatResponse
				err = json.Unmarshal(msg.Data, &heartBeatResponse)
				if err != nil {
					continue
				}
				results[c] = heartBeatResponse.StreamIds
			}

			pc.mu.Lock()
			for c, ids := range results {
				log.Info().Msgf("Ids = %s", ids)
				pc.activeStreamIds[c] = ids
			}
			pc.mu.Unlock()

		}
	}
}

func (pc *ProducerClient) WriteResponse(conn *ConnectionDetails, obj interface{}, writer *Writer) (int, error) {
	pc.mu.Lock()
	streamIds, active := pc.activeStreamIds[conn.ConnId]

	for _, v := range streamIds {
		if v == conn.StreamId {
			active = true
		}

	}
	pc.mu.Unlock()

	if !active {
		return 0, fmt.Errorf("stream id is now closed")
	}

	return writer.WriteObject(obj)
}

func (pc *ProducerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    pc.Conn,
		subject: subject,
	}
}
