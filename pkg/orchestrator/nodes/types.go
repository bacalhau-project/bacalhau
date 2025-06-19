//go:generate mockgen --source types.go --destination mocks.go --package nodes

// Package nodes provides node lifecycle and health management for distributed compute clusters.
//
// The package implements a node manager that handles node registration, health monitoring,
// state tracking, and resource management. It maintains both in-memory state for fast access
// and persistent storage for durability.
//
// Key features:
//   - Node lifecycle management (registration, approval/rejection, deletion)
//   - Health monitoring via heartbeats
//   - Connection state tracking
//   - Resource capacity tracking
//   - Event notifications for state changes
//
// Basic usage:
//
//	manager, err := nodes.NewManager(nodes.ManagerParams{
//	    Store: store,
//	    NodeDisconnectedAfter: 5 * time.Minute,
//	})
//	if err != nil {
//	    return err
//	}
//
//	if err := manager.Start(ctx); err != nil {
//	    return err
//	}
//
//	// Register connection state handler
//	manager.OnConnectionStateChange(func(event NodeConnectionEvent) {
//	    log.Printf("Node %s changed from %s to %s",
//	        event.NodeID, event.Previous, event.Current)
//	})
package nodes

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

// Manager defines the interface for node lifecycle and health management.
// It provides operations for node registration, state updates, and queries.
type Manager interface {
	// Start initializes the manager and begins background tasks.
	// It loads existing node states and starts health monitoring.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the manager and its background tasks.
	// It ensures state is persisted before stopping.
	Stop(ctx context.Context) error

	// Running returns whether the manager is currently active.
	Running() bool

	// Handshake handles initial node registration or reconnection.
	// It validates the node and establishes its initial state.
	Handshake(ctx context.Context, request messages.HandshakeRequest) (messages.HandshakeResponse, error)

	// UpdateNodeInfo updates a node's information and capabilities.
	// The node must be registered and not rejected.
	UpdateNodeInfo(ctx context.Context, request messages.UpdateNodeInfoRequest) (messages.UpdateNodeInfoResponse, error)

	// ShutdownNotice handles a node's graceful shutdown notification.
	// It updates sequence numbers and marks the node as cleanly disconnected.
	ShutdownNotice(ctx context.Context, request ExtendedShutdownNoticeRequest) (messages.ShutdownNoticeResponse, error)

	// Heartbeat processes a node's heartbeat message and updates its state.
	// It returns the last known sequence numbers for synchronization.
	Heartbeat(ctx context.Context, request ExtendedHeartbeatRequest) (messages.HeartbeatResponse, error)

	// ApproveNode approves a node for cluster participation.
	// Returns error if node is already approved or not found.
	ApproveNode(ctx context.Context, nodeID string) error

	// RejectNode rejects a node from cluster participation.
	// Returns error if node is already rejected or not found.
	RejectNode(ctx context.Context, nodeID string) error

	// DeleteNode removes a node from the cluster.
	// Returns error if node is not found.
	DeleteNode(ctx context.Context, nodeID string) error

	// OnConnectionStateChange registers a handler for node connection state changes.
	OnConnectionStateChange(handler ConnectionStateChangeHandler)

	Lookup

	Tracker
}

type Lookup interface {
	// Get retrieves a node's state by exact ID.
	Get(ctx context.Context, nodeID string) (models.NodeState, error)

	// GetByPrefix retrieves a node's state by ID prefix.
	GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error)

	// List returns all nodes matching the given filters.
	List(ctx context.Context, filters ...NodeStateFilter) ([]models.NodeState, error)
}

type Tracker interface {
	// GetConnectedNodesCount returns the number of currently connected nodes.
	GetConnectedNodesCount() int
}

// Store defines the interface for persistent node state storage.
type Store interface {
	Lookup

	// Put stores a node's state.
	Put(ctx context.Context, state models.NodeState) error

	// Delete removes a node's state.
	Delete(ctx context.Context, nodeID string) error
}

// NodeStateFilter defines a function type for filtering node states.
type NodeStateFilter func(models.NodeState) bool

// HealthyNodeFilter is a filter that returns only nodes that are healthy and connected.
func HealthyNodeFilter(state models.NodeState) bool {
	return state.ConnectionState.Status == models.NodeStates.CONNECTED
}

// NodeConnectionEvent represents a change in a node's connection state.
type NodeConnectionEvent struct {
	// NodeID is the identifier of the node whose state changed
	NodeID string

	// Previous is the previous connection state
	Previous models.NodeConnectionState

	// Current is the new connection state
	Current models.NodeConnectionState

	// Timestamp is when the state change occurred
	Timestamp time.Time
}

// ConnectionStateChangeHandler defines a function type for handling connection state changes.
type ConnectionStateChangeHandler func(NodeConnectionEvent)

// ExtendedHeartbeatRequest represents a heartbeat message with additional metadata.
type ExtendedHeartbeatRequest struct {
	messages.HeartbeatRequest

	// LastComputeSeqNum is the last processed compute message sequence
	LastComputeSeqNum uint64
}

// ExtendedShutdownNoticeRequest represents a shutdown message with additional metadata.
type ExtendedShutdownNoticeRequest struct {
	messages.ShutdownNoticeRequest

	// LastComputeSeqNum is the last processed compute message sequence
	LastComputeSeqNum uint64
}
