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
	return h.publisher.Publish(ctx, Heartbeat{NodeID: h.nodeID, Sequence: sequence})
}

var _ Client = (*HeartbeatClient)(nil)
