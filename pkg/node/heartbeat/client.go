package heartbeat

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
)

type HeartbeatClient struct {
	publisher ncl.Publisher
	nodeID    string
}

func NewClient(nodeID string, publisher ncl.Publisher) *HeartbeatClient {
	return &HeartbeatClient{publisher: publisher, nodeID: nodeID}
}

func (h *HeartbeatClient) SendHeartbeat(ctx context.Context, sequence uint64) error {
	message := ncl.NewMessage(Heartbeat{NodeID: h.nodeID, Sequence: sequence})
	return h.publisher.Publish(ctx, message)
}

var _ Client = (*HeartbeatClient)(nil)
