package stream

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"sync"
	"time"
)

type ProducerClientParams struct {
	Conn *nats.Conn
}

type ConnectionDetails struct {
	ConnId              string
	StreamId            string
	HeartBeatRequestSub string
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
			pc.mu.Lock()
			data, _ := json.Marshal(pc.activeStreamIds)
			pc.mu.Unlock()

			for c, v := range pc.activeConnHeartBeatRequestSubjects {

				log.Info().Msgf("HEART BEAT REQUEST TO %s", v)
				msg, err := pc.Conn.Request(v, data, 5*time.Second)
				if err != nil {
					pc.activeStreamIds[c] = pc.activeStreamIds[c][:0]
					continue
				}
				log.Ctx(ctxWithCancel).Info().Msgf("Heart Beat Response = %s", string(msg.Data))
			}

		}
	}
}

func (pc *ProducerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    pc.Conn,
		subject: subject,
	}
}
