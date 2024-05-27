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
	Config StreamProducerClientConfig
}

type ProducerClient struct {
	Conn *nats.Conn
	mu   sync.RWMutex // Protects access to activeStreamInfo and activeConnHeartBeatRequestSubjects

	// A map of ConsumerID to StreamId that are active
	activeStreamInfo map[string]map[string]StreamInfo
	// A map of ConsumerID to the subject where a heartBeatRequest needs to be sent.
	activeConnHeartBeatRequestSubjects map[string]string
	heartBeatCancelFunc                context.CancelFunc
	config                             StreamProducerClientConfig
}

func NewProducerClient(ctx context.Context, params ProducerClientParams) (*ProducerClient, error) {
	nc := &ProducerClient{
		Conn:                               params.Conn,
		activeStreamInfo:                   make(map[string]map[string]StreamInfo),
		activeConnHeartBeatRequestSubjects: make(map[string]string),
		config:                             params.Config,
	}

	go nc.heartBeat(ctx)

	return nc, nil
}

func (pc *ProducerClient) AddStream(
	consumerID string,
	streamID string,
	requestSub string,
	heartBeatRequestSub string,
) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	streamInfo := StreamInfo{
		ID:         streamID,
		RequestSub: requestSub,
		CreatedAt:  time.Now(),
	}

	if pc.activeStreamInfo[consumerID] == nil {
		pc.activeStreamInfo[consumerID] = make(map[string]StreamInfo)
	}

	if _, ok := pc.activeStreamInfo[consumerID][streamID]; ok {
		return fmt.Errorf("cannot create request with same streamId %s again", streamID)
	}

	pc.activeStreamInfo[consumerID][streamID] = streamInfo
	pc.activeConnHeartBeatRequestSubjects[consumerID] = heartBeatRequestSub

	return nil
}

func (pc *ProducerClient) RemoveStream(consumerID string, streamID string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	activeStreamIdsForConn, ok := pc.activeStreamInfo[consumerID]
	if !ok {
		return
	}

	if _, ok := activeStreamIdsForConn[streamID]; !ok {
		return
	}

	delete(activeStreamIdsForConn, streamID)

	if len(activeStreamIdsForConn) == 0 {
		delete(pc.activeStreamInfo, consumerID)
		delete(pc.activeConnHeartBeatRequestSubjects, consumerID)
	}
}

func (pc *ProducerClient) heartBeat(ctx context.Context) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	pc.heartBeatCancelFunc = cancel

	ticker := time.NewTicker(pc.config.HeartBeatIntervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithCancel.Done():
			log.Ctx(ctxWithCancel).Debug().Msg("Heart beat for NATS based streaming producer client cancelled.")
			return
		case <-ticker.C:

			nonActiveStreamIds := make(map[string][]string)

			for c, v := range pc.activeConnHeartBeatRequestSubjects {
				// Create an empty slice for activeStreamIds
				activeStreamIds := make(map[string][]string)

				if streamInfoMap, ok := pc.activeStreamInfo[c]; ok {
					for _, streamInfo := range streamInfoMap {
						activeStreamIds[streamInfo.RequestSub] = append(activeStreamIds[streamInfo.RequestSub], streamInfo.ID)
					}
				}

				heartBeatRequest := HeartBeatRequest{
					ActiveStreamIds: activeStreamIds,
				}

				data, err := json.Marshal(heartBeatRequest)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while parsing heart beat request in NATS streaming producer client")
					continue
				}

				msg, err := pc.Conn.Request(v, data, pc.config.HeartBeatRequestTimeout)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while sending heart beat request from NATS streaming producer client")
					nonActiveStreamIds[c] = []string{}
					continue
				}

				var heartBeatResponse ConsumerHeartBeatResponse
				err = json.Unmarshal(msg.Data, &heartBeatResponse)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while  parsing heart beat response from NATS streaming consumer client")
					continue
				}

				nonActiveStreamIds[c] = getStringList(heartBeatResponse.NonActiveStreamIds)
			}

			pc.updateActiveStreamInfo(nonActiveStreamIds)
		}
	}
}

func (pc *ProducerClient) WriteResponse(
	consumerID string,
	streamID string,
	obj interface{},
	writer *Writer,
) (int, error) {
	pc.mu.Lock()
	streamIds, active := pc.activeStreamInfo[consumerID]

	if !active {
		return 0, fmt.Errorf("no stream ids exist to write for consumerId=%s", consumerID)
	}
	pc.mu.Unlock()

	_, active = streamIds[streamID]
	if !active {
		return 0, fmt.Errorf("streamId %s is now closed", streamID)
	}

	return writer.WriteObject(obj)
}

func (pc *ProducerClient) updateActiveStreamInfo(nonActiveStreamIds map[string][]string) {
	// Create a map to store the streams that need to be deleted.
	streamsToDelete := make(map[string][]string)

	for connID, nonActiveIdsList := range nonActiveStreamIds {
		// Create a map for quick lookup of non-active streams.
		nonActiveMap := make(map[string]bool)
		for _, id := range nonActiveIdsList {
			nonActiveMap[id] = true
		}

		if streamInfo, ok := pc.activeStreamInfo[connID]; ok {
			for streamID := range streamInfo {
				// If the stream is not active, store it for deletion.
				if nonActiveMap[streamID] {
					streamsToDelete[connID] = append(streamsToDelete[connID], streamID)
				}
			}
		}
	}

	// Delete all inactive streams with minimal lock contention.
	pc.mu.Lock()
	for connID, streams := range streamsToDelete {
		for _, streamID := range streams {
			delete(pc.activeStreamInfo[connID], streamID)
		}

		// If after deletion, there's no stream left for this connection, delete the connection
		if len(pc.activeStreamInfo[connID]) == 0 {
			delete(pc.activeStreamInfo, connID)
		}
	}
	pc.mu.Unlock()
}

func (pc *ProducerClient) NewWriter(subject string) *Writer {
	return &Writer{
		conn:    pc.Conn,
		subject: subject,
	}
}

func getStringList(m map[string][]string) []string {
	var result []string
	for _, v := range m {
		result = append(result, v...)
	}
	return result
}
