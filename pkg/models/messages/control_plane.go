package messages

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// HandshakeRequest is exchanged during initial connection
type HandshakeRequest struct {
	NodeInfo               models.NodeInfo `json:"NodeInfo"`
	StartTime              time.Time       `json:"StartTime"`
	LastOrchestratorSeqNum uint64          `json:"LastOrchestratorSeqNum"` // Last seq received from orchestrator
}

// HandshakeResponse is sent in response to handshake requests
type HandshakeResponse struct {
	Accepted                   bool   `json:"accepted"`
	Reason                     string `json:"reason,omitempty"`
	LastComputeSeqNum          uint64 `json:"LastComputeSeqNum"`      // Last seq received from compute node
	StartingOrchestratorSeqNum uint64 `json:"LastOrchestratorSeqNum"` // Seq to start sending to compute node
}

type HeartbeatRequest struct {
	NodeID                 string           `json:"NodeID"`
	AvailableCapacity      models.Resources `json:"AvailableCapacity"`
	QueueUsedCapacity      models.Resources `json:"QueueUsedCapacity"`
	LastOrchestratorSeqNum uint64           `json:"LastOrchestratorSeqNum"` // Last seq received from orchestrator
}

type HeartbeatResponse struct {
	LastComputeSeqNum uint64 `json:"LastComputeSeqNum"` // Last seq received from compute node
}

// UpdateNodeInfoRequest is used to update the node info
type UpdateNodeInfoRequest struct {
	NodeInfo models.NodeInfo `json:"NodeInfo"`
}
type UpdateNodeInfoResponse struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

// ShutdownNoticeRequest tells the orchestrator that this node is shutting down
type ShutdownNoticeRequest struct {
	NodeID                 string `json:"NodeID"`
	Reason                 string `json:"Reason,omitempty"`
	LastOrchestratorSeqNum uint64 `json:"LastOrchestratorSeqNum"` // Last seq received from orchestrator
}

// ShutdownNoticeResponse sends any final instructions back to the shutting-down node
type ShutdownNoticeResponse struct {
	LastComputeSeqNum uint64 `json:"LastComputeSeqNum"` // Last seq received from compute node
}
