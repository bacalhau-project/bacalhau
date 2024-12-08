package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	messages "github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type HeartbeatClient struct {
	publisher ncl.Publisher
	nodeID    string
}

func NewHeartbeatClient(nodeID string, publisher ncl.Publisher) (*HeartbeatClient, error) {
	return &HeartbeatClient{publisher: publisher, nodeID: nodeID}, nil
}

func (h *HeartbeatClient) SendHeartbeat(ctx context.Context, sequence uint64) error {
	heartbeat := messages.Heartbeat{NodeID: h.nodeID, Sequence: sequence}

	// Send the heartbeat to current and legacy topics
	message := envelope.NewMessage(heartbeat)
	return h.publisher.Publish(ctx, ncl.NewPublishRequest(message))
}

func (h *HeartbeatClient) Close(ctx context.Context) error {
	return nil
}
