package heartbeat

import (
	"context"
	"errors"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	natsPubSub "github.com/bacalhau-project/bacalhau/pkg/nats/pubsub"
)

type HeartbeatClient struct {
	legacyPublisher *natsPubSub.PubSub[messages.Heartbeat]
	publisher       ncl.Publisher
	nodeID          string
}

func NewClient(conn *nats.Conn, nodeID string, publisher ncl.Publisher) (*HeartbeatClient, error) {
	legacyPublisher, err := natsPubSub.NewPubSub[messages.Heartbeat](natsPubSub.PubSubParams{
		Subject: legacyHeartbeatTopic,
		Conn:    conn,
	})
	if err != nil {
		return nil, err
	}

	return &HeartbeatClient{publisher: publisher, nodeID: nodeID, legacyPublisher: legacyPublisher}, nil
}

func (h *HeartbeatClient) SendHeartbeat(ctx context.Context, sequence uint64) error {
	heartbeat := messages.Heartbeat{NodeID: h.nodeID, Sequence: sequence}

	// Send the heartbeat to current and legacy topics
	message := ncl.NewMessage(heartbeat)
	err := h.publisher.Publish(ctx, ncl.NewPublishRequest(message))
	err = errors.Join(err, h.legacyPublisher.Publish(ctx, heartbeat))
	return err
}

func (h *HeartbeatClient) Close(ctx context.Context) error {
	return h.legacyPublisher.Close(ctx)
}

var _ Client = (*HeartbeatClient)(nil)
