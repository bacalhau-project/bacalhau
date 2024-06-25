package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

type ProducerClientParams struct {
	Conn   *nats.Conn
	Config StreamProducerClientConfig
}

type ProducerClient struct {
	Conn *nats.Conn
	mu   sync.RWMutex // Protects access to activeConsumers

	activeConsumers     map[string]consumerInfo
	heartBeatCancelFunc context.CancelFunc
	config              StreamProducerClientConfig
}

type consumerInfo struct {
	// Heartbeat request subject to which consumer info subscribes to respond
	// with non-active stream ids
	HeartbeatRequestSub string
	// A map holding information about active streams alive at consumer
	ActiveStreamInfo map[string]StreamInfo
}

func (c *consumerInfo) getActiveStreamIds() []string {
	return lo.Keys(c.ActiveStreamInfo)
}

func (c *consumerInfo) getActiveStreamIdsByRequestSubject() map[string][]string {
	activeStreamIdsByReqSubj := make(map[string][]string)

	for streamID, streamInfo := range c.ActiveStreamInfo {
		activeStreamIdsByReqSubj[streamInfo.RequestSub] = append(activeStreamIdsByReqSubj[streamInfo.RequestSub], streamID)
	}
	return activeStreamIdsByReqSubj
}

func NewProducerClient(ctx context.Context, params ProducerClientParams) (*ProducerClient, error) {
	nc := &ProducerClient{
		Conn:            params.Conn,
		activeConsumers: make(map[string]consumerInfo),
		config:          params.Config,
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

	if _, ok := pc.activeConsumers[consumerID]; !ok {
		pc.activeConsumers[consumerID] = consumerInfo{
			HeartbeatRequestSub: heartBeatRequestSub,
			ActiveStreamInfo:    make(map[string]StreamInfo),
		}
	}

	if _, ok := pc.activeConsumers[consumerID].ActiveStreamInfo[streamID]; ok {
		return fmt.Errorf("cannot create request with same streamId %s again", streamID)
	}

	streamInfo := StreamInfo{
		ID:         streamID,
		RequestSub: requestSub,
		CreatedAt:  time.Now(),
		Cancel:     cancelFunc,
	}

	pc.activeConsumers[consumerID].ActiveStreamInfo[streamID] = streamInfo
	return nil
}

func (pc *ProducerClient) RemoveStream(consumerID string, streamID string) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	activeStreamIdsForConn := pc.activeConsumers[consumerID].ActiveStreamInfo
	if activeStreamIdsForConn == nil {
		return
	}

	if _, ok := activeStreamIdsForConn[streamID]; !ok {
		return
	}

	delete(activeStreamIdsForConn, streamID)

	if len(activeStreamIdsForConn) == 0 {
		delete(pc.activeConsumers, consumerID)
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

			for c, v := range pc.activeConsumers {
				heartBeatRequest := HeartBeatRequest{
					ActiveStreamIds: v.getActiveStreamIdsByRequestSubject(),
				}

				data, err := json.Marshal(heartBeatRequest)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("error while parsing heart beat request in NATS streaming producer client")
					continue
				}

				msg, err := pc.Conn.Request(v.HeartbeatRequestSub, data, pc.config.HeartBeatRequestTimeout)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("heartbeat request to consumer client timed out")
					nonActiveStreamIds[c] = append(nonActiveStreamIds[c], v.getActiveStreamIds()...)
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

		if consumer, ok := pc.activeConsumers[connID]; ok {
			for streamID := range consumer.ActiveStreamInfo {
				if nonActiveMap[streamID] {
					streamInfo := consumer.ActiveStreamInfo[streamID]
					streamInfo.Cancel()
					delete(consumer.ActiveStreamInfo, streamID)
				}
			}
			// If after deletion, there's no stream left for this connection, delete the connection
			if len(consumer.ActiveStreamInfo) == 0 {
				delete(pc.activeConsumers, connID)
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
