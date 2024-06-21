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
	cancelFunc context.CancelFunc,
) error {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	streamInfo := StreamInfo{
		ID:         streamID,
		RequestSub: requestSub,
		CreatedAt:  time.Now(),
		Cancel:     cancelFunc,
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
			pc.mu.RLock()

			for c, v := range pc.activeConnHeartBeatRequestSubjects {
				// Create an empty slice for activeStreamIdsByReqSubj
				activeStreamIdsByReqSubj := make(map[string][]string)
				var activeStreamIds []string

				if streamInfoMap, ok := pc.activeStreamInfo[c]; ok {
					for streamId, streamInfo := range streamInfoMap {
						activeStreamIds = append(activeStreamIds, streamId)
						activeStreamIdsByReqSubj[streamInfo.RequestSub] = append(activeStreamIdsByReqSubj[streamInfo.RequestSub], streamInfo.ID)
					}
				}

				heartBeatRequest := HeartBeatRequest{
					ActiveStreamIds: activeStreamIdsByReqSubj,
				}

				data, err := json.Marshal(heartBeatRequest)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while parsing heart beat request in NATS streaming producer client")
					continue
				}

				msg, err := pc.Conn.Request(v, data, pc.config.HeartBeatRequestTimeout)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("heartbeat request to consumer client timed out")
					nonActiveStreamIds[c] = append(nonActiveStreamIds[c], activeStreamIds...)
					continue
				}

				var heartBeatResponse ConsumerHeartBeatResponse
				err = json.Unmarshal(msg.Data, &heartBeatResponse)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while parsing heart beat response from NATS streaming consumer client")
					continue
				}

				nonActiveStreamIdsFromConsumer := getStringList(heartBeatResponse.NonActiveStreamIds)
				if len(nonActiveStreamIdsFromConsumer) != 0 {
					nonActiveStreamIds[c] = append(nonActiveStreamIds[c], nonActiveStreamIdsFromConsumer...)
				}
			}

			pc.mu.RUnlock()
			if len(nonActiveStreamIds) != 0 {
				pc.updateActiveStreamInfo(nonActiveStreamIds)
			}
		}
	}
}

func (pc *ProducerClient) updateActiveStreamInfo(nonActiveStreamIds map[string][]string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	for connID, nonActiveIdsList := range nonActiveStreamIds {
		// Create a map for quick lookup of non-active streams.
		nonActiveMap := make(map[string]bool)
		for _, id := range nonActiveIdsList {
			nonActiveMap[id] = true
		}

		if streamInfo, ok := pc.activeStreamInfo[connID]; ok {
			for streamID := range streamInfo {
				if nonActiveMap[streamID] {
					streamInfo := pc.activeStreamInfo[connID][streamID]
					streamInfo.Cancel()
					delete(pc.activeStreamInfo[connID], streamID)
				}
			}
			// If after deletion, there's no stream left for this connection, delete the connection
			if len(pc.activeStreamInfo[connID]) == 0 {
				delete(pc.activeStreamInfo, connID)
				delete(pc.activeConnHeartBeatRequestSubjects, connID)
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

func getStringList(m map[string][]string) []string {
	var result []string
	for _, v := range m {
		result = append(result, v...)
	}
	return result
}
