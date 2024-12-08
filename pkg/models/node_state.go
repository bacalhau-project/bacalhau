package models

import (
	"time"
)

// NodeState contains metadata about the state of a node on the network. Requester nodes maintain a NodeState for
// each node they are aware of. The NodeState represents a Requester nodes view of another node on the network.
type NodeState struct {
	// Durable node information
	Info       NodeInfo            `json:"Info"`
	Membership NodeMembershipState `json:"Membership"`

	// Connection and messaging state
	ConnectionState ConnectionState `json:"ConnectionState"`
}

// ConnectionState tracks node's connectivity and messaging state
type ConnectionState struct {
	// Connection status
	Status NodeConnectionState `json:"Status"` // Connected, Disconnected, etc.

	// Last successful heartbeat timestamp
	LastHeartbeat time.Time `json:"LastHeartbeat"`

	// Message sequencing for reliable delivery
	LastComputeSeqNum      uint64 `json:"LastComputeSeqNum"`      // Last seq received from compute node
	LastOrchestratorSeqNum uint64 `json:"LastOrchestratorSeqNum"` // Last seq received from orchestrator

	// Connection tracking
	ConnectedSince    time.Time `json:"ConnectedSince"`
	DisconnectedSince time.Time `json:"DisconnectedSince,omitempty"`
	LastError         string    `json:"LastError,omitempty"`
}

func (s *NodeState) IsConnected() bool {
	return s.ConnectionState.Status == NodeStates.CONNECTED
}
