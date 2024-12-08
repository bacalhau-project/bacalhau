package legacy

import "github.com/bacalhau-project/bacalhau/pkg/models"

const (
	HeartbeatMessageType = "heartbeat"
)

// Heartbeat represents a heartbeat message from a specific node.
// It contains the node ID and the sequence number of the heartbeat
// which is monotonically increasing (reboots aside). We do not
// use timestamps on the client, we rely solely on the server-side
// time to avoid clock drift issues.
type Heartbeat struct {
	NodeID   string
	Sequence uint64
}

type RegisterRequest struct {
	Info models.NodeInfo
}

type RegisterResponse struct {
	Accepted bool
	Reason   string
}

type UpdateInfoRequest struct {
	Info models.NodeInfo
}

type UpdateInfoResponse struct {
	Accepted bool
	Reason   string
}

type UpdateResourcesRequest struct {
	NodeID            string
	AvailableCapacity models.Resources
	QueueUsedCapacity models.Resources
}

type UpdateResourcesResponse struct{}
