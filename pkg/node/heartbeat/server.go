package heartbeat

import (
	"context"
	"errors"
	"reflect"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
)

const (
	// heartbeatCheckFrequencyFactor is the factor by which the disconnectedAfter time
	// will be divided to determine the frequency of the heartbeat check.
	heartbeatCheckFrequencyFactor = 3
	minHeartbeatCheckFrequency    = 1 * time.Second
	maxHeartbeatCheckFrequency    = 30 * time.Second
)

type HeartbeatServerParams struct {
	NodeID                string
	Client                *nats.Conn
	Clock                 clock.Clock
	NodeDisconnectedAfter time.Duration
}

type HeartbeatServer struct {
	nodeID            string
	clock             clock.Clock
	legacySubscriber  *natsPubSub.PubSub[messages.Heartbeat]
	pqueue            *collections.HashedPriorityQueue[string, TimestampedHeartbeat]
	livenessMap       *concurrency.StripedMap[models.NodeConnectionState]
	checkFrequency    time.Duration
	disconnectedAfter time.Duration
}

type TimestampedHeartbeat struct {
	messages.Heartbeat
	Timestamp int64
}

func NewServer(params HeartbeatServerParams) (*HeartbeatServer, error) {
	legacySubscriber, err := natsPubSub.NewPubSub[messages.Heartbeat](natsPubSub.PubSubParams{
		Subject: legacyHeartbeatTopic,
		Conn:    params.Client,
	})
	if err != nil {
		return nil, err
	}

	pqueue := collections.NewHashedPriorityQueue[string, TimestampedHeartbeat](
		func(h TimestampedHeartbeat) string {
			return h.NodeID
		},
	)

	// If no clock was specified, use the real time clock
	clk := params.Clock
	if clk == nil {
		clk = clock.New()
	}

	// We'll set the frequency of the heartbeat check to be 1/3 of the disconnected
	heartbeatCheckFrequency := params.NodeDisconnectedAfter / heartbeatCheckFrequencyFactor
	if heartbeatCheckFrequency < minHeartbeatCheckFrequency {
		heartbeatCheckFrequency = minHeartbeatCheckFrequency
	} else if heartbeatCheckFrequency > maxHeartbeatCheckFrequency {
		heartbeatCheckFrequency = maxHeartbeatCheckFrequency
	}

	return &HeartbeatServer{
		nodeID:            params.NodeID,
		clock:             clk,
		legacySubscriber:  legacySubscriber,
		pqueue:            pqueue,
		livenessMap:       concurrency.NewStripedMap[models.NodeConnectionState](0), // no particular stripe count for now
		checkFrequency:    heartbeatCheckFrequency,
		disconnectedAfter: params.NodeDisconnectedAfter,
	}, nil
}

func (h *HeartbeatServer) Start(ctx context.Context) error {
	if err := h.legacySubscriber.Subscribe(ctx, h); err != nil {
		return err
	}

	tickerStartCh := make(chan struct{})

	go func(ctx context.Context) {
		defer func() {
			if err := h.legacySubscriber.Close(ctx); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
				log.Ctx(ctx).Error().Err(err).Msg("Error during heartbeat server shutdown")
			} else {
				log.Ctx(ctx).Debug().Msg("Heartbeat server shutdown")
			}
		}()

		ticker := h.clock.Ticker(h.checkFrequency)
		tickerStartCh <- struct{}{}
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.checkQueue(ctx)
			}
		}
	}(ctx)

	// Wait for the ticker to be created before returning
	<-tickerStartCh
	log.Ctx(ctx).Debug().Msg("Heartbeat server started")

	return nil
}

// checkQueue will check the queue for old heartbeats that might make a node's
// liveness either unhealthy or unknown, and will update the node's status accordingly.
// This method is not thread-safe and should be called from a single goroutine.
func (h *HeartbeatServer) checkQueue(ctx context.Context) {
	// Calculate the timestamp threshold for considering a node as disconnected
	disconnectedUnder := h.clock.Now().Add(-h.disconnectedAfter).UTC().Unix()

	for {
		// Peek at the next (oldest) item in the queue
		peek := h.pqueue.Peek()

		// If the queue is empty, we're done
		if peek == nil {
			break
		}

		// If the oldest item is recent enough, we're done
		log.Ctx(ctx).Trace().
			Dur("LastHeartbeatAge", h.clock.Now().Sub(time.Unix(peek.Value.Timestamp, 0))).
			Msgf("Peeked at %+v", peek)
		if peek.Value.Timestamp >= disconnectedUnder {
			break
		}

		// Dequeue the item and mark the node as disconnected
		item := h.pqueue.Dequeue()
		if item == nil || item.Value.Timestamp >= disconnectedUnder {
			// This should never happen, but we'll check just in case
			log.Warn().Msgf("Unexpected item dequeued: %+v didn't match previously peeked item: %+v", item, peek)
			continue
		}

		if item.Value.NodeID == h.nodeID {
			// We don't want to mark ourselves as disconnected
			continue
		}

		log.Ctx(ctx).Debug().
			Str("NodeID", item.Value.NodeID).
			Int64("LastHeartbeat", item.Value.Timestamp).
			Dur("LastHeartbeatAge", h.clock.Now().Sub(time.Unix(item.Value.Timestamp, 0))).
			Msg("Marking node as disconnected")
		h.markNodeAs(item.Value.NodeID, models.NodeStates.DISCONNECTED)
	}
}

// markNode will mark a node as being in a certain state. This will be used to update the node's
// info to include the liveness state.
func (h *HeartbeatServer) markNodeAs(nodeID string, state models.NodeConnectionState) {
	h.livenessMap.Put(nodeID, state)
}

// UpdateNode will add the liveness for specific nodes to their NodeInfo
func (h *HeartbeatServer) UpdateNodeInfo(state *models.NodeState) {
	if state.Info.NodeID == h.nodeID {
		// We don't want to mark ourselves as disconnected
		state.Connection = models.NodeStates.CONNECTED
	} else if liveness, ok := h.livenessMap.Get(state.Info.NodeID); ok {
		state.Connection = liveness
	} else {
		// We've never seen this, so we'll mark it as unknown
		state.Connection = models.NodeStates.DISCONNECTED
	}
}

// FilterNodeInfos will return only those NodeInfos that have the requested liveness
func (h *HeartbeatServer) FilterNodeInfos(nodeInfos []*models.NodeInfo, state models.NodeConnectionState) []*models.NodeInfo {
	result := make([]*models.NodeInfo, 0)
	for _, nodeInfo := range nodeInfos {
		if liveness, ok := h.livenessMap.Get(nodeInfo.NodeID); ok {
			if liveness == state {
				result = append(result, nodeInfo)
			}
		}
	}
	return result
}

// RemoveNode will handle removing the liveness for a specific node. This is useful when a node
// is removed from the cluster.
func (h *HeartbeatServer) RemoveNode(nodeID string) {
	h.livenessMap.Delete(nodeID)
}

func (h *HeartbeatServer) ShouldProcess(ctx context.Context, message *envelope.Message) bool {
	return message.Metadata.Get(envelope.KeyMessageType) == HeartbeatMessageType
}

// Handle will handle a message received through the legacy heartbeat topic
func (h *HeartbeatServer) Handle(ctx context.Context, heartbeat messages.Heartbeat) error {
	timestamp := h.clock.Now().UTC().Unix()
	th := TimestampedHeartbeat{Heartbeat: heartbeat, Timestamp: timestamp}
	log.Ctx(ctx).Trace().Msgf("Enqueueing heartbeat from %s with seq %d. %+v", th.NodeID, th.Sequence, th)

	// We'll enqueue the heartbeat message with the current timestamp in reverse priority so that
	// older heartbeats are dequeued first.
	h.pqueue.Enqueue(th, -timestamp)
	h.markNodeAs(heartbeat.NodeID, models.NodeStates.HEALTHY)
	return nil
}

// HandleMessage will handle a message received through ncl and will call the Handle method
func (h *HeartbeatServer) HandleMessage(ctx context.Context, message *envelope.Message) error {
	heartbeat, ok := message.Payload.(*messages.Heartbeat)
	if !ok {
		return envelope.NewErrUnexpectedPayloadType(
			reflect.TypeOf(messages.Heartbeat{}).String(), reflect.TypeOf(message.Payload).String())
	}
	return h.Handle(ctx, *heartbeat)
}

var _ ncl.MessageHandler = (*HeartbeatServer)(nil)
var _ pubsub.Subscriber[messages.Heartbeat] = (*HeartbeatServer)(nil)
