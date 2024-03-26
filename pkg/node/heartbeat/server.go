package heartbeat

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"
)

type HeartbeatServer struct {
	subscription *natsPubSub.PubSub[Heartbeat]
	pqueue       *collections.HashedPriorityQueue[string, Heartbeat]
}

func NewServer(conn *nats.Conn) (*HeartbeatServer, error) {
	subParams := natsPubSub.PubSubParams{
		Subject: heartbeatTopic,
		Conn:    conn,
	}

	subscription, err := natsPubSub.NewPubSub[Heartbeat](subParams)
	if err != nil {
		return nil, err
	}

	pqueue := collections.NewHashedPriorityQueue[string, Heartbeat](
		func(h Heartbeat) string {
			return h.NodeID
		},
	)

	return &HeartbeatServer{subscription: subscription, pqueue: pqueue}, nil
}

func (h *HeartbeatServer) Start(ctx context.Context) error {
	if err := h.subscription.Subscribe(ctx, h); err != nil {
		return err
	}

	go func(ctx context.Context) {
		log.Ctx(ctx).Info().Msg("Heartbeat server started")
		<-ctx.Done()
		_ = h.subscription.Close(ctx)
		log.Ctx(ctx).Info().Msg("Heartbeat server shutdown")
	}(ctx)

	go func(ctx context.Context) {
		ticker := time.NewTicker(heartbeatQueueCheckFrequency)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// We'll iterate over the priority queue and check if the last heartbeat
				// received from each node is older than 10 seconds. If it is, we'll
				// consider the node as dead.
			}
		}
	}(ctx)

	return nil
}

func (h *HeartbeatServer) Handle(ctx context.Context, message Heartbeat) error {
	log.Ctx(ctx).Trace().Msgf("heartbeat received from %s", message.NodeID)

	timestamp := time.Now().UTC().Unix()

	if h.pqueue.Contains(message.NodeID) {
		// If we think we already have a heartbeat from this node, we'll update the
		// timestamp of the entry so it is re-prioritized in the queue by dequeuing
		// and re-enqueuing it (this will ensure it is heapified correctly).
		result := h.pqueue.DequeueWhere(func(item Heartbeat) bool {
			return item.NodeID == message.NodeID
		})

		log.Ctx(ctx).Trace().Msgf("Re-enqueueing heartbeat from %s", message.NodeID)
		h.pqueue.Enqueue(result.Value, timestamp)
	} else {
		log.Ctx(ctx).Trace().Msgf("Enqueueing heartbeat from %s", message.NodeID)

		// We'll enqueue the heartbeat message with the current timestamp. The older
		// the entry, the lower the timestamp (trending to 0) and the higher the priority.
		h.pqueue.Enqueue(message, timestamp)
	}

	return nil
}

var _ pubsub.Subscriber[Heartbeat] = (*HeartbeatServer)(nil)
