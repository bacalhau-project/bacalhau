package heartbeat

import (
	"context"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
)

type HeartbeatServerParams struct {
	Client                *nats.Conn
	Topic                 string
	Clock                 clock.Clock
	CheckFrequency        time.Duration
	NodeDisconnectedAfter time.Duration
}

type HeartbeatServer struct {
	clock             clock.Clock
	subscription      *natsPubSub.PubSub[Heartbeat]
	pqueue            *collections.HashedPriorityQueue[string, TimestampedHeartbeat]
	livenessMap       *concurrency.StripedMap[models.NodeConnectionState]
	checkFrequency    time.Duration
	disconnectedAfter time.Duration
}

type TimestampedHeartbeat struct {
	Heartbeat
	Timestamp int64
}

func NewServer(params HeartbeatServerParams) (*HeartbeatServer, error) {
	subParams := natsPubSub.PubSubParams{
		Subject: params.Topic,
		Conn:    params.Client,
	}

	subscription, err := natsPubSub.NewPubSub[Heartbeat](subParams)
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

	return &HeartbeatServer{
		clock:             clk,
		subscription:      subscription,
		pqueue:            pqueue,
		livenessMap:       concurrency.NewStripedMap[models.NodeConnectionState](0), // no particular stripe count for now
		checkFrequency:    params.CheckFrequency,
		disconnectedAfter: params.NodeDisconnectedAfter,
	}, nil
}

func (h *HeartbeatServer) Start(ctx context.Context) error {
	if err := h.subscription.Subscribe(ctx, h); err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msg("Heartbeat server started")

	tickerStartCh := make(chan struct{})

	go func(ctx context.Context) {
		defer func() {
			if err := h.subscription.Close(ctx); err != nil {
				log.Ctx(ctx).Error().Err(err).Msg("Error during heartbeat server shutdown")
			} else {
				log.Ctx(ctx).Info().Msg("Heartbeat server shutdown")
			}
		}()

		ticker := h.clock.Ticker(h.checkFrequency)
		tickerStartCh <- struct{}{}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.CheckQueue(ctx)
			}
		}
	}(ctx)

	// Wait for the ticker to be created before returning
	<-tickerStartCh

	return nil
}

// CheckQueue will check the queue for old heartbeats that might make a node's
// liveness either unhealthy or unknown, and will update the node's status accordingly.
func (h *HeartbeatServer) CheckQueue(ctx context.Context) {
	// These are the timestamps, below which we'll consider the item in one of those two
	// states
	nowStamp := h.clock.Now().UTC().Unix()
	disconnectedUnder := nowStamp - int64(h.disconnectedAfter.Seconds())

	for {
		// Dequeue anything older than the unknown timestamp
		item := h.pqueue.DequeueWhere(func(item TimestampedHeartbeat) bool {
			return item.Timestamp < disconnectedUnder
		})

		// We haven't found anything old enough yet. We can stop the loop and wait
		// for the next cycle.
		if item == nil {
			break
		}

		if item.Value.Timestamp < disconnectedUnder {
			h.markNodeAs(item.Value.NodeID, models.NodeStates.DISCONNECTED)
		}
	}
}

// markNode will mark a node as being in a certain state. This will be used to update the node's
// info to include the liveness state.
func (h *HeartbeatServer) markNodeAs(nodeID string, state models.NodeConnectionState) {
	h.livenessMap.Put(nodeID, state)
}

// UpdateNode will add the liveness for specific nodes to their NodeInfo
func (h *HeartbeatServer) UpdateNodeInfo(state *models.NodeState) {
	if liveness, ok := h.livenessMap.Get(state.Info.NodeID); ok {
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

func (h *HeartbeatServer) Handle(ctx context.Context, message Heartbeat) error {
	log.Ctx(ctx).Trace().Msgf("heartbeat received from %s", message.NodeID)

	timestamp := h.clock.Now().UTC().Unix()

	if h.pqueue.Contains(message.NodeID) {
		// If we think we already have a heartbeat from this node, we'll update the
		// timestamp of the entry so it is re-prioritized in the queue by dequeuing
		// and re-enqueuing it (this will ensure it is heapified correctly).
		result := h.pqueue.DequeueWhere(func(item TimestampedHeartbeat) bool {
			return item.NodeID == message.NodeID
		})

		if result == nil {
			log.Ctx(ctx).Warn().Msgf("consistency error in heartbeat heap, node %s not found", message.NodeID)
			return nil
		}

		log.Ctx(ctx).Trace().Msgf("Re-enqueueing heartbeat from %s", message.NodeID)
		result.Value.Timestamp = timestamp
		h.pqueue.Enqueue(result.Value, timestamp)
	} else {
		log.Ctx(ctx).Trace().Msgf("Enqueueing heartbeat from %s", message.NodeID)

		// We'll enqueue the heartbeat message with the current timestamp. The older
		// the entry, the lower the timestamp (trending to 0) and the higher the priority.
		h.pqueue.Enqueue(TimestampedHeartbeat{Heartbeat: message, Timestamp: timestamp}, timestamp)
	}

	h.markNodeAs(message.NodeID, models.NodeStates.HEALTHY)

	return nil
}

var _ pubsub.Subscriber[Heartbeat] = (*HeartbeatServer)(nil)
