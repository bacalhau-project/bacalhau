package heartbeat

import (
	"context"

	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/rs/zerolog/log"

	"github.com/nats-io/nats.go"
)

type HeartbeatClient struct {
	publisher *natsPubSub.PubSub[Heartbeat]
	nodeID    string
}

func NewClient(conn *nats.Conn, nodeID string) (*HeartbeatClient, error) {
	subParams := natsPubSub.PubSubParams{
		Subject: heartbeatTopic,
		Conn:    conn,
	}

	publisher, err := natsPubSub.NewPubSub[Heartbeat](subParams)
	if err != nil {
		return nil, err
	}

	return &HeartbeatClient{publisher: publisher, nodeID: nodeID}, nil
}

func (h *HeartbeatClient) Start(ctx context.Context) error {
	// Waits until we are cancelled and then closes the publisher
	<-ctx.Done()
	return h.publisher.Close(ctx)
}

func (h *HeartbeatClient) SendHeartbeat(ctx context.Context, sequence uint64) error {
	log.Ctx(ctx).Trace().Msgf("sending heartbeat seq: %d", sequence)
	return h.Publish(ctx, Heartbeat{NodeID: h.nodeID, Sequence: sequence})
}

func (h *HeartbeatClient) Publish(ctx context.Context, message Heartbeat) error {
	return h.publisher.Publish(ctx, message)
}

var _ pubsub.Publisher[Heartbeat] = (*HeartbeatClient)(nil)
