package heartbeat

import (
	"context"

	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"

	"github.com/nats-io/nats.go"
)

type HeartbeatClient struct {
	publisher *natsPubSub.PubSub[Heartbeat]
	nodeID    string
}

func NewClient(conn *nats.Conn, nodeID string, topic string) (*HeartbeatClient, error) {
	subParams := natsPubSub.PubSubParams{
		Subject: topic,
		Conn:    conn,
	}

	publisher, err := natsPubSub.NewPubSub[Heartbeat](subParams)
	if err != nil {
		return nil, err
	}

	return &HeartbeatClient{publisher: publisher, nodeID: nodeID}, nil
}

func (h *HeartbeatClient) SendHeartbeat(ctx context.Context, sequence uint64) error {
	return h.Publish(ctx, Heartbeat{NodeID: h.nodeID, Sequence: sequence})
}

func (h *HeartbeatClient) Publish(ctx context.Context, message Heartbeat) error {
	return h.publisher.Publish(ctx, message)
}

func (h *HeartbeatClient) Close(ctx context.Context) error {
	return h.publisher.Close(ctx)
}

var _ pubsub.Publisher[Heartbeat] = (*HeartbeatClient)(nil)
