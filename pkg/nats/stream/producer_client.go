package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type ProducerClientParams struct {
	Conn   *nats.Conn
	ID     string
	Config StreamProducerClientConfig
}

type ProducerClient struct {
	Conn *nats.Conn
	ID   string
	mu   sync.RWMutex // Protects access to activeStreamInfo and activeConnHeartBeatRequestSubjects

	// A map of ConnID to StreamId that are active
	activeStreamInfo map[string]map[string]StreamInfo
	// A map of ConnID to the subject where a heartBeatRequest needs to be sent.
	activeConnHeartBeatRequestSubjects map[string]string
	heartBeatCancelFunc                context.CancelFunc
	config                             StreamProducerClientConfig
}

func NewProducerClient(params ProducerClientParams) (*ProducerClient, error) {
	nc := &ProducerClient{
		Conn:                               params.Conn,
		ID:                                 params.ID,
		activeStreamInfo:                   make(map[string]map[string]StreamInfo),
		activeConnHeartBeatRequestSubjects: make(map[string]string),
		config:                             params.Config,
	}

	go nc.heartBeat(context.Background())

	return nc, nil
}

func (pc *ProducerClient) AddConnDetails(ctx context.Context, connDetails *ConnectionDetails) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	streamInfo := StreamInfo{
		ID:        connDetails.StreamID,
		CreatedAt: time.Now(),
	}

	if pc.activeStreamInfo[connDetails.ConnID] == nil {
		pc.activeStreamInfo[connDetails.ConnID] = make(map[string]StreamInfo)
	}

	if _, ok := pc.activeStreamInfo[connDetails.ConnID][connDetails.StreamID]; ok {
		return fmt.Errorf("cannot create request with same streamId %s again", connDetails.StreamID)
	}

	pc.activeStreamInfo[connDetails.ConnID][connDetails.StreamID] = streamInfo
	pc.activeConnHeartBeatRequestSubjects[connDetails.ConnID] = connDetails.HeartBeatRequestSub

	return nil
}

func (pc *ProducerClient) RemoveConnDetails(connDetails *ConnectionDetails) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	activeStreamIdsForConn, ok := pc.activeStreamInfo[connDetails.ConnID]
	if !ok {
		return
	}

	if _, ok := activeStreamIdsForConn[connDetails.StreamID]; !ok {
		return
	}

	delete(activeStreamIdsForConn, connDetails.StreamID)

	if len(activeStreamIdsForConn) == 0 {
		delete(pc.activeStreamInfo, connDetails.ConnID)
		delete(pc.activeConnHeartBeatRequestSubjects, connDetails.ConnID)
	}

	if len(pc.activeStreamInfo) == 0 && pc.heartBeatCancelFunc != nil {
		pc.heartBeatCancelFunc()
		pc.heartBeatCancelFunc = nil
	}
}

func (pc *ProducerClient) heartBeat(ctx context.Context) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	pc.heartBeatCancelFunc = cancel

	ticker := time.NewTicker(pc.config.HeartBeatConfig.HeartBeatIntervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithCancel.Done():
			log.Ctx(ctxWithCancel).Debug().Msgf("Heart beat for producer client %s", pc.ID)
			return
		case <-ticker.C:

			nonActiveStreamIds := make(map[string][]string)

			for c, v := range pc.activeConnHeartBeatRequestSubjects {
				// Create an empty slice for activeStreamIds
				var activeStreamIds []string

				if streamInfoMap, ok := pc.activeStreamInfo[c]; ok {
					for _, streamInfo := range streamInfoMap {
						activeStreamIds = append(activeStreamIds, streamInfo.ID)
					}
				}

				heartBeatRequest := HeartBeatRequest{
					ProducerConnID:  pc.ID,
					ActiveStreamIds: activeStreamIds,
				}

				data, err := json.Marshal(heartBeatRequest)
				if err != nil {
					log.Err(err)
					continue
				}

				msg, err := pc.Conn.Request(v, data, pc.config.HeartBeatConfig.HeartBeatRequestTimeout)
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
		if v.ID == conn.StreamID {
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
	updates := make(map[string]map[string]StreamInfo)

	for connID, nonActiveIdsList := range nonActiveStreamIds {
		nonActiveMap := make(map[string]bool)
		for _, id := range nonActiveIdsList {
			nonActiveMap[id] = true
		}

		updatedStreams := make(map[string]StreamInfo)
		for streamID, streamInfo := range pc.activeStreamInfo[connID] {
			if _, found := nonActiveMap[streamID]; !found ||
				time.Since(streamInfo.CreatedAt) <= pc.config.StreamCancellationBufferDuration {
				updatedStreams[streamID] = streamInfo
			}
		}

		if len(updatedStreams) > 0 {
			updates[connID] = updatedStreams
		}
	}

	pc.mu.Lock()
	for c, s := range updates {
		pc.activeStreamInfo[c] = s
	}
	pc.mu.Unlock()
}

func (pc *ProducerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    pc.Conn,
		subject: subject,
	}
}
