package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
	"sync"
	"time"
)

type ProducerClientParams struct {
	Conn            *nats.Conn
	HeartBeatConfig HeartBeatConfig
}

type ProducerClient struct {
	Conn *nats.Conn
	Id   string
	mu   sync.RWMutex // Protects access to activeStreamInfo and activeConnHeartBeatRequestSubjects

	activeStreamInfo                   map[string][]StreamInfo // A map of ConnID to StreamId that are active
	activeConnHeartBeatRequestSubjects map[string]string       // A map of ConnID to the subject where a heart beat request should be sent
	heartBeatCancelFunc                context.CancelFunc
}

func NewProducerClient(params ProducerClientParams) (*ProducerClient, error) {
	nc := &ProducerClient{
		Conn:                               params.Conn,
		Id:                                 params.Conn.Opts.Name,
		activeStreamInfo:                   make(map[string][]StreamInfo),
		activeConnHeartBeatRequestSubjects: make(map[string]string),
	}

	return nc, nil
}

func (pc *ProducerClient) AddConnDetails(ctx context.Context, connDetails *ConnectionDetails) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	streamInfo := StreamInfo{
		Id:        connDetails.StreamID,
		CreatedAt: time.Now(),
	}
	pc.activeStreamInfo[connDetails.ConnID] = append(pc.activeStreamInfo[connDetails.ConnID], streamInfo)
	pc.activeConnHeartBeatRequestSubjects[connDetails.ConnID] = connDetails.HeartBeatRequestSub

	if pc.heartBeatCancelFunc == nil {
		go pc.heartBeat(ctx)
	}

}

func (pc *ProducerClient) RemoveConnDetails(connDetails *ConnectionDetails) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	activeStreamIdsForConn, ok := pc.activeStreamInfo[connDetails.ConnID]
	if !ok {
		return
	}

	for i, v := range activeStreamIdsForConn {
		if v.Id == connDetails.StreamID {
			activeStreamIdsForConn = append(activeStreamIdsForConn[:i], activeStreamIdsForConn[i+1:]...)
		}
	}

	if len(activeStreamIdsForConn) == 0 {
		delete(pc.activeStreamInfo, connDetails.ConnID)
		delete(pc.activeConnHeartBeatRequestSubjects, connDetails.StreamID)
	}

	if len(pc.activeStreamInfo) == 0 && pc.heartBeatCancelFunc != nil {
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

			nonActiveStreamIds := make(map[string][]string)

			for c, v := range pc.activeConnHeartBeatRequestSubjects {
				heartBeatRequest := HeartBeatRequest{
					ProducerConnID: pc.Id,
					ActiveStreamIds: lo.Map(pc.activeStreamInfo[c],
						func(streamInfo StreamInfo, _ int) string { return streamInfo.Id }),
				}

				data, err := json.Marshal(heartBeatRequest)
				if err != nil {
					log.Err(err)
					continue
				}

				msg, err := pc.Conn.Request(v, data, 5*time.Second)
				if err != nil {
					nonActiveStreamIds[c] = []string{}
					continue
				}

				var heartBeatResponse ConsumerHeartBeatResponse
				err = json.Unmarshal(msg.Data, &heartBeatResponse)
				if err != nil {
					continue
				}
				nonActiveStreamIds[c] = heartBeatResponse.NonActiveStreamIds
			}

			pc.updateActiveStreamInfo(nonActiveStreamIds)
		}
	}
}

func (pc *ProducerClient) WriteResponse(conn *ConnectionDetails, obj interface{}, writer *Writer) (int, error) {
	pc.mu.Lock()
	streamIds, active := pc.activeStreamInfo[conn.ConnID]

	for _, v := range streamIds {
		if v.Id == conn.StreamID {
			active = true
		}

	}
	pc.mu.Unlock()

	if !active {
		return 0, fmt.Errorf("stream id is now closed")
	}

	return writer.WriteObject(obj)
}

func (pc *ProducerClient) updateActiveStreamInfo(nonActiveStreamIds map[string][]string) {
	updates := make(map[string][]StreamInfo)
	for c, nonActiveIdsList := range nonActiveStreamIds {
		nonActiveMap := make(map[string]bool)
		for _, id := range nonActiveIdsList {
			nonActiveMap[id] = true
		}

		update := make([]StreamInfo, 0)
		for _, streamInfo := range pc.activeStreamInfo[c] {
			if _, found := nonActiveMap[streamInfo.Id]; found && time.Since(streamInfo.CreatedAt) > 10*time.Second {
				continue
			}

			update = append(update, streamInfo)
		}

		updates[c] = update
	}

	pc.mu.Lock()
	for c, update := range updates {
		pc.activeStreamInfo[c] = update
	}
	pc.mu.Unlock()

}

func (pc *ProducerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    pc.Conn,
		subject: subject,
	}
}
