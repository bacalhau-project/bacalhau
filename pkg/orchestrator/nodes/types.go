//go:generate mockgen --source types.go --destination mocks.go --package nodes
package nodes

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type Manager interface {
	Lookup

	Handshake(ctx context.Context, request messages.HandshakeRequest) (messages.HandshakeResponse, error)
	UpdateNodeInfo(ctx context.Context, request messages.UpdateNodeInfoRequest) (messages.UpdateNodeInfoResponse, error)
	Heartbeat(ctx context.Context, request ExtendedHeartbeatRequest) (messages.HeartbeatResponse, error)

	ApproveNode(ctx context.Context, nodeID string) error
	RejectNode(ctx context.Context, nodeID string) error
	DeleteNode(ctx context.Context, nodeID string) error

	OnConnectionStateChange(handler ConnectionStateChangeHandler)

	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Lookup interface {
	// Get returns the node info for the given node ID.
	Get(ctx context.Context, nodeID string) (models.NodeState, error)

	// GetByPrefix returns the node info for the given node ID.
	// Supports both full and short node IDs.
	GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error)

	// List returns a list of nodes
	List(ctx context.Context, filters ...NodeStateFilter) ([]models.NodeState, error)
}

type Store interface {
	Lookup

	// Put adds a node info to the repo.
	Put(ctx context.Context, nodeInfo models.NodeState) error

	// Delete deletes a node info from the repo.
	Delete(ctx context.Context, nodeID string) error
}

// NodeStateFilter is a function that filters node state
// when listing nodes. It returns true if the node state
// should be returned, and false if the node state should
// be ignored.
type NodeStateFilter func(models.NodeState) bool

// ConnectionStateChangeHandler is called when a node's connection state changes
type ConnectionStateChangeHandler func(NodeConnectionEvent)

// NodeConnectionEvent represents a connection state change
type NodeConnectionEvent struct {
	NodeID    string
	Previous  models.NodeConnectionState
	Current   models.NodeConnectionState
	Timestamp time.Time
}

type ExtendedHeartbeatRequest struct {
	messages.HeartbeatRequest
	LastComputeSeqNum uint64
}
