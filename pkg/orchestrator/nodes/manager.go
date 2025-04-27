package nodes

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/analytics"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

const (
	// heartbeatCheckFrequencyFactor is the factor by which the disconnectedAfter time
	// will be divided to determine the frequency of the heartbeat check.
	heartbeatCheckFrequencyFactor = 3

	// defaultPersistInterval is the default interval for state persistence
	defaultPersistInterval = 5 * time.Minute
	defaultPersistTimeout  = 10 * time.Second
	defaultShutdownTimeout = 10 * time.Second

	// Minimum and maximum heartbeat check frequencies to ensure reasonable bounds
	minHeartbeatCheckFrequency = 1 * time.Second
	maxHeartbeatCheckFrequency = 30 * time.Second

	// DefaultAnalyticsInterval is the default interval for publishing analytics
	defaultAnalyticsInterval = 1 * time.Hour
)

// nodesManager handles node lifecycle, health checking, and state management.
// It maintains both in-memory state for fast access and persistent storage
// for durability. The manager provides:
//   - Node registration and handshake handling
//   - Health monitoring via heartbeats
//   - Connection state tracking with notifications
//   - Resource capacity monitoring
//   - Background state persistence
//
// Thread safety is ensured through sync.Map for live state and proper mutex
// usage for control operations. Background tasks handle health checks and
// state persistence with configurable intervals.
type nodesManager struct {
	// Core dependencies
	store            Store                   // Persistent storage for node states
	eventstore       watcher.EventStore      // Event store for sequence number tracking
	nodeInfoProvider models.NodeInfoProvider // Provides node information for self registration
	clock            clock.Clock             // Time source (can be mocked for testing)

	// Configuration
	defaultApprovalState    models.NodeMembershipState // Initial membership state for new nodes
	heartbeatCheckFrequency time.Duration              // How often to check node health
	disconnectedAfter       time.Duration              // Time after which to mark nodes as disconnected
	persistInterval         time.Duration              // For periodic persistence
	persistTimeout          time.Duration
	shutdownTimeout         time.Duration
	analyticsInterval       time.Duration // How often to publish analytics

	// Runtime state
	liveState      *sync.Map // Thread-safe map of nodeID -> trackedLiveState
	connectedNodes int64     // Atomic counter for connected nodes

	// Background task management
	tasks   sync.WaitGroup // Tracks running background tasks
	stopCh  chan struct{}  // Signals background tasks to stop
	running bool           // Whether the manager is currently running
	mu      sync.RWMutex   // Protects running state

	// Event handlers
	handlers struct {
		sync.RWMutex
		connectionState []ConnectionStateChangeHandler
	}
}

// ManagerParams holds configuration for creating a new node manager.
type ManagerParams struct {
	// Store provides persistent storage for node states
	Store Store

	// Clock is the time source (defaults to real clock if nil)
	Clock clock.Clock

	// NodeInfoProvider provides node information for self registration
	NodeInfoProvider models.NodeInfoProvider

	// NodeDisconnectedAfter is how long to wait before marking nodes as disconnected
	NodeDisconnectedAfter time.Duration

	// HealthCheckFrequency is how often to check node health (optional)
	HealthCheckFrequency time.Duration

	// ManualApproval determines if nodes require manual approval
	ManualApproval bool

	// PersistInterval is how often to persist state changes (optional)
	PersistInterval time.Duration

	// PersistTimeout is the timeout for persistence operations (optional)
	PersistTimeout time.Duration

	// ShutdownTimeout is the timeout for graceful shutdown (optional)
	ShutdownTimeout time.Duration

	// AnalyticsInterval is how often to publish analytics (optional)
	AnalyticsInterval time.Duration

	// EventStore provides storage for events so that node manager can assign
	// new nodes with latest sequence number in the store
	EventStore watcher.EventStore
}

// trackedLiveState holds the runtime state for an active node.
// This includes current connection status and resource utilization.
type trackedLiveState struct {
	connectionState   models.ConnectionState
	availableCapacity models.Resources
	queueUsedCapacity models.Resources
}

// NewManager creates a new nodesManager with the given configuration.
// It initializes the manager but does not start background tasks - call Start() for that.
func NewManager(params ManagerParams) (Manager, error) {
	if params.Clock == nil {
		params.Clock = clock.New()
	}

	// Determine initial approval state based on configuration
	defaultApprovalState := models.NodeMembership.APPROVED
	if params.ManualApproval {
		defaultApprovalState = models.NodeMembership.PENDING
	}

	// Calculate health check frequency within bounds if not explicitly set
	heartbeatCheckFrequency := params.HealthCheckFrequency
	if heartbeatCheckFrequency == 0 {
		heartbeatCheckFrequency = params.NodeDisconnectedAfter / heartbeatCheckFrequencyFactor
		if heartbeatCheckFrequency < minHeartbeatCheckFrequency {
			heartbeatCheckFrequency = minHeartbeatCheckFrequency
		} else if heartbeatCheckFrequency > maxHeartbeatCheckFrequency {
			heartbeatCheckFrequency = maxHeartbeatCheckFrequency
		}
	}

	if params.PersistInterval == 0 {
		params.PersistInterval = defaultPersistInterval
	}
	if params.PersistTimeout == 0 {
		params.PersistTimeout = defaultPersistTimeout
	}
	if params.ShutdownTimeout == 0 {
		params.ShutdownTimeout = defaultShutdownTimeout
	}

	if params.AnalyticsInterval == 0 {
		params.AnalyticsInterval = defaultAnalyticsInterval
	}

	if err := errors.Join(
		validate.NotNil(params.Store, "store required"),
		validate.NotNil(params.EventStore, "event store required"),
		validate.NotNil(params.NodeInfoProvider, "node info provider required"),
	); err != nil {
		return nil, fmt.Errorf("node manager invalid params: %w", err)
	}

	return &nodesManager{
		store:                   params.Store,
		eventstore:              params.EventStore,
		nodeInfoProvider:        params.NodeInfoProvider,
		clock:                   params.Clock,
		liveState:               &sync.Map{},
		defaultApprovalState:    defaultApprovalState,
		heartbeatCheckFrequency: heartbeatCheckFrequency,
		disconnectedAfter:       params.NodeDisconnectedAfter,
		persistInterval:         params.PersistInterval,
		persistTimeout:          params.PersistTimeout,
		shutdownTimeout:         params.ShutdownTimeout,
		analyticsInterval:       params.AnalyticsInterval,
		stopCh:                  make(chan struct{}),
	}, nil
}

// Start initializes the manager and begins background tasks.
// It launches health checking and state persistence routines.
// The manager will monitor the provided context and initiate
// shutdown if it is cancelled.
//
// Returns error if already running or fails to initialize.
func (n *nodesManager) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return bacerrors.New("node manager already running").
			WithCode(bacerrors.BadRequestError).
			WithComponent(errComponent)
	}

	// Initialize clean state
	n.liveState = &sync.Map{}
	n.stopCh = make(chan struct{})

	// self register the node's info in the store
	if err := n.selfRegister(ctx); err != nil {
		return err
	}

	// Start background tasks
	n.startBackgroundTask("health-check", n.healthCheckLoop)
	n.startBackgroundTask("state-persistence", n.persistenceLoop)
	n.startBackgroundTask("analytics", n.analyticsLoop)

	// Monitor parent context for cancellation
	go func() {
		select {
		case <-ctx.Done():
			// Parent context was cancelled, trigger stop
			log.Debug().Msg("Parent context cancelled, stopping node manager")
			stopCtx, cancel := context.WithTimeout(context.Background(), n.shutdownTimeout)
			defer cancel()
			if stopErr := n.Stop(stopCtx); stopErr != nil {
				log.Error().Err(stopErr).Msg("Failed to stop node manager gracefully")
			}
		case <-n.stopCh:
			// Normal shutdown, nothing to do
			return
		}
	}()

	n.running = true
	return nil
}

// Stop gracefully shuts down the manager and its background tasks.
// It ensures final state persistence and waits for tasks to complete
// up to the configured shutdown timeout.
//
// Returns nil if successfully stopped or already stopped,
// context.Err() if shutdown times out.
func (n *nodesManager) Stop(ctx context.Context) error {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return nil
	}
	n.running = false
	close(n.stopCh)
	n.mu.Unlock()

	// Wait for background tasks with timeout
	done := make(chan struct{})
	go func() {
		n.tasks.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Running returns whether the manager is currently active.
func (n *nodesManager) Running() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// startBackgroundTask launches a new background task with proper cleanup.
// Tasks should respect context cancellation and the stop channel.
func (n *nodesManager) startBackgroundTask(name string, fn func()) {
	n.tasks.Add(1)
	go func() {
		defer n.tasks.Done()
		fn()
	}()
}

// healthCheckLoop runs periodic health checks on all nodes.
// It runs on the configured check frequency and marks nodes
// as disconnected if they haven't sent a heartbeat within
// the disconnect timeout period.
func (n *nodesManager) healthCheckLoop() {
	ticker := n.clock.Ticker(n.heartbeatCheckFrequency)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.checkNodeHealth()
		}
	}
}

// checkNodeHealth checks all nodes and marks them as disconnected if they've timed out.
// It preserves sequence numbers and other state while updating the connection status.
func (n *nodesManager) checkNodeHealth() {
	// Track unhealthy nodes with their observed state
	type unhealthyNode struct {
		id    string
		state *trackedLiveState
	}
	var unhealthyNodes []unhealthyNode

	// First pass - identify unhealthy nodes and capture their state
	n.liveState.Range(func(key, value interface{}) bool {
		nodeID := key.(string)
		state := value.(*trackedLiveState)

		if n.isNodeDisconnected(state.connectionState) {
			unhealthyNodes = append(unhealthyNodes, unhealthyNode{
				id:    nodeID,
				state: state,
			})
		}
		return true
	})

	for _, node := range unhealthyNodes {
		// Mark node as disconnected
		log.Info().Str("node", node.id).
			Time("lastHeartbeat", node.state.connectionState.LastHeartbeat).
			Msg("Marking node as disconnected")

		// Try to update live state only if it hasn't changed since we checked it
		newConnectionState := node.state.connectionState
		newConnectionState.Status = models.NodeStates.DISCONNECTED
		newConnectionState.DisconnectedSince = n.clock.Now().UTC()
		newConnectionState.LastError = "heartbeat timeout"

		newState := &trackedLiveState{
			connectionState:   newConnectionState,
			availableCapacity: models.Resources{},
			queueUsedCapacity: models.Resources{},
		}

		if !n.liveState.CompareAndSwap(node.id, node.state, newState) {
			log.Debug().Str("node", node.id).Msg("Node state changed since health check, skipping update")
			continue
		}

		n.notifyConnectionStateChange(NodeConnectionEvent{
			NodeID:    node.id,
			Previous:  models.NodeStates.CONNECTED,
			Current:   models.NodeStates.DISCONNECTED,
			Timestamp: n.clock.Now().UTC(),
		})
	}
}

// persistenceLoop periodically persists live state changes to storage.
// It runs on the configured persist interval and ensures durability
// of connection state and resource tracking.
func (n *nodesManager) persistenceLoop() {
	ticker := n.clock.Ticker(n.persistInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			n.persistLiveState() // Final persistence before stopping
			return
		case <-ticker.C:
			n.persistLiveState()
		}
	}
}

func (n *nodesManager) persistLiveState() {
	ctx, cancel := context.WithTimeout(context.Background(), n.persistTimeout)
	defer cancel()

	n.liveState.Range(func(key, value interface{}) bool {
		nodeID := key.(string)
		liveState := value.(*trackedLiveState)

		// Get existing state from store
		state, err := n.store.Get(ctx, nodeID)
		if err != nil {
			log.Error().Err(err).Str("node", nodeID).Msg("Failed to get node state during persistence")
			return true
		}

		// Persist only if connection state has changed
		if state.ConnectionState.Status == liveState.connectionState.Status &&
			state.ConnectionState.LastOrchestratorSeqNum == liveState.connectionState.LastOrchestratorSeqNum &&
			state.ConnectionState.LastComputeSeqNum == liveState.connectionState.LastComputeSeqNum {
			return true
		}

		// Update with live state
		state.ConnectionState = liveState.connectionState
		state.Info.ComputeNodeInfo.AvailableCapacity = liveState.availableCapacity
		state.Info.ComputeNodeInfo.QueueUsedCapacity = liveState.queueUsedCapacity

		// Persist to store
		if err = n.store.Put(ctx, state); err != nil {
			log.Error().Err(err).Str("node", nodeID).Msg("Failed to persist node state")
		}
		return true
	})
}

// analyticsLoop periodically publishes analytics about connected nodes
func (n *nodesManager) analyticsLoop() {
	if !analytics.IsEnabled() {
		log.Trace().Msg("Analytics is disabled, skipping nodes analytics")
		return
	}

	ticker := n.clock.Ticker(n.analyticsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.publishAnalytics()
		}
	}
}

// publishAnalytics collects and publishes analytics about connected nodes
func (n *nodesManager) publishAnalytics() {
	// Get all nodes
	ctx := context.Background()
	nodes, err := n.List(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list nodes for analytics")
		return
	}

	// Create and emit analytics event
	analytics.Emit(analytics.NewNodeInfosEvent(nodes))
}

// Handshake handles initial node registration or reconnection.
// For new nodes, it:
//   - Validates the node type
//   - Creates initial node state
//   - Assigns default approval status
//
// For existing nodes, it:
//   - Verifies the node isn't rejected
//   - Restores previous membership status
//   - Updates connection state
//
// Returns HandshakeResponse with acceptance status and reason.
// The LastComputeSeqNum is included for message ordering.
func (n *nodesManager) Handshake(
	ctx context.Context, request messages.HandshakeRequest) (messages.HandshakeResponse, error) {
	log.Debug().Msgf("handshake request received with info %+v", request.NodeInfo)

	existingConnectionState := models.NodeStates.DISCONNECTED
	isReconnect := false
	var existing models.NodeState

	// Check if node is already registered, and if so, if it was rejected
	existing, err := n.Get(ctx, request.NodeInfo.ID())
	if err == nil {
		if existing.Membership == models.NodeMembership.REJECTED {
			return messages.HandshakeResponse{
				Accepted: false,
				Reason:   "node has been rejected",
			}, nil
		}

		isReconnect = true
		existingConnectionState = existing.ConnectionState.Status

		// Check if node connection status is outdated
		if n.isNodeDisconnected(existing.ConnectionState) {
			existingConnectionState = models.NodeStates.DISCONNECTED
		}
	}

	// Validate the node is compute type
	if !request.NodeInfo.IsComputeNode() {
		return messages.HandshakeResponse{
			Accepted: false,
			Reason:   "node is not a compute node",
		}, nil
	}

	// Create new/updated node state
	state := models.NodeState{
		Info:       request.NodeInfo,
		Membership: n.defaultApprovalState,
		ConnectionState: models.ConnectionState{
			Status:         models.NodeStates.CONNECTED,
			ConnectedSince: n.clock.Now().UTC(),
			LastHeartbeat:  n.clock.Now().UTC(),
		},
	}

	if isReconnect {
		state.Membership = existing.Membership
		state.ConnectionState.LastComputeSeqNum = existing.ConnectionState.LastComputeSeqNum
	}

	// Resolve where the node should start receiving messages from
	state.ConnectionState.LastOrchestratorSeqNum, err = n.resolveStartingOrchestratorSeqNum(ctx, isReconnect, existing)
	if err != nil {
		return messages.HandshakeResponse{}, fmt.Errorf("failed to resolve starting sequence number: %w", err)
	}

	if err = n.store.Put(ctx, state); err != nil {
		return messages.HandshakeResponse{}, err
	}

	// Store live state for resource tracking
	n.liveState.Store(state.Info.ID(), &trackedLiveState{
		connectionState:   state.ConnectionState,
		availableCapacity: state.Info.ComputeNodeInfo.AvailableCapacity,
		queueUsedCapacity: state.Info.ComputeNodeInfo.QueueUsedCapacity,
	})

	n.notifyConnectionStateChange(NodeConnectionEvent{
		NodeID:    request.NodeInfo.ID(),
		Previous:  existingConnectionState,
		Current:   state.ConnectionState.Status,
		Timestamp: n.clock.Now().UTC(),
	})

	log.Info().Msgf("handshake successful with node %s", request.NodeInfo.ID())

	reason := "node registered"
	if isReconnect {
		reason = "node reconnected"
	}
	return messages.HandshakeResponse{
		Accepted:                   true,
		Reason:                     reason,
		LastComputeSeqNum:          state.ConnectionState.LastComputeSeqNum,
		StartingOrchestratorSeqNum: state.ConnectionState.LastOrchestratorSeqNum,
	}, nil
}

// UpdateNodeInfo updates a node's information and capabilities.
// The node must:
//   - Be already registered (handshake completed)
//   - Not be in rejected state
//
// Returns UpdateNodeInfoResponse with acceptance status and reason.

func (n *nodesManager) UpdateNodeInfo(
	ctx context.Context, request messages.UpdateNodeInfoRequest) (messages.UpdateNodeInfoResponse, error) {
	existing, err := n.Get(ctx, request.NodeInfo.ID())
	if err != nil {
		if bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError) {
			// return an error that a handshake is first required
			return messages.UpdateNodeInfoResponse{}, NewErrHandshakeRequired(request.NodeInfo.ID())
		}
		return messages.UpdateNodeInfoResponse{}, err
	}

	if existing.Membership == models.NodeMembership.REJECTED {
		return messages.UpdateNodeInfoResponse{
			Accepted: false,
			Reason:   "node registration rejected",
		}, nil
	}

	existing.Info = request.NodeInfo
	if err = n.store.Put(ctx, existing); err != nil {
		return messages.UpdateNodeInfoResponse{}, err
	}

	return messages.UpdateNodeInfoResponse{
		Accepted: true,
	}, nil
}

// Heartbeat processes a node's heartbeat message and updates its state.
// It updates:
//   - Last heartbeat timestamp
//   - Message sequence numbers
//   - Resource capacities
//
// The update is retried up to 3 times on concurrent modification.
// Returns HeartbeatResponse with the last known compute sequence number.
func (n *nodesManager) Heartbeat(
	ctx context.Context, request ExtendedHeartbeatRequest) (messages.HeartbeatResponse, error) {
	// Retry loop for concurrent updates, such as handshakes or health checks
	maxConcurrentAttempts := 3
	for i := 0; i < maxConcurrentAttempts; i++ {
		// Get existing live state
		existingEntry, ok := n.liveState.Load(request.NodeID)
		if !ok {
			return messages.HeartbeatResponse{}, NewErrHandshakeRequired(request.NodeID)
		}

		existing := existingEntry.(*trackedLiveState)
		if existing.connectionState.Status != models.NodeStates.CONNECTED {
			return messages.HeartbeatResponse{}, NewErrHandshakeRequired(request.NodeID)
		}

		// updated connection state
		updated := existing.connectionState
		updated.LastHeartbeat = n.clock.Now().UTC()
		n.updateSequenceNumbers(&updated, request.LastOrchestratorSeqNum, request.LastComputeSeqNum)

		// Store updated state back if no concurrent modification
		if !n.liveState.CompareAndSwap(request.NodeID, existing, &trackedLiveState{
			connectionState:   updated,
			availableCapacity: request.AvailableCapacity,
			queueUsedCapacity: request.QueueUsedCapacity,
		}) {
			continue
		}

		return messages.HeartbeatResponse{
			LastComputeSeqNum: updated.LastComputeSeqNum,
		}, nil
	}
	return messages.HeartbeatResponse{}, NewErrConcurrentModification()
}

// ShutdownNotice processes a shutdown notification from a node and updates its state.
// It updates:
//   - Final sequence numbers
//   - Connection state to disconnected
//   - Preserves the sequence numbers in persistent storage
//
// Returns ShutdownNoticeResponse with the last sequence number processed from that node.
func (n *nodesManager) ShutdownNotice(
	ctx context.Context, request ExtendedShutdownNoticeRequest) (messages.ShutdownNoticeResponse, error) {
	// Get existing live state
	existingEntry, ok := n.liveState.Load(request.NodeID)
	if !ok {
		return messages.ShutdownNoticeResponse{}, NewErrHandshakeRequired(request.NodeID)
	}

	existing := existingEntry.(*trackedLiveState)
	if existing.connectionState.Status != models.NodeStates.CONNECTED {
		return messages.ShutdownNoticeResponse{}, NewErrHandshakeRequired(request.NodeID)
	}

	// Update connection state with final sequence numbers
	updated := existing.connectionState
	updated.Status = models.NodeStates.DISCONNECTED
	updated.DisconnectedSince = n.clock.Now().UTC()
	n.updateSequenceNumbers(&updated, request.LastOrchestratorSeqNum, request.LastComputeSeqNum)
	updated.LastError = "graceful shutdown"

	// Attempt atomic update
	if !n.liveState.CompareAndSwap(request.NodeID, existingEntry, &trackedLiveState{
		connectionState:   updated,
		availableCapacity: models.Resources{}, // Clear capacity since node is shutting down
		queueUsedCapacity: models.Resources{},
	}) {
		return messages.ShutdownNoticeResponse{}, NewErrConcurrentModification()
	}

	log.Info().
		Str("node", request.NodeID).
		Str("reason", request.Reason).
		Uint64("lastOrchestratorSeq", updated.LastOrchestratorSeqNum).
		Uint64("lastComputeSeq", updated.LastComputeSeqNum).
		Msg("Node shutdown notice received")

	// Notify about state change
	n.notifyConnectionStateChange(NodeConnectionEvent{
		NodeID:    request.NodeID,
		Previous:  models.NodeStates.CONNECTED,
		Current:   models.NodeStates.DISCONNECTED,
		Timestamp: updated.DisconnectedSince,
	})

	return messages.ShutdownNoticeResponse{
		LastComputeSeqNum: updated.LastComputeSeqNum,
	}, nil
}

// ApproveNode approves a node for cluster participation.
// The node must be in PENDING state. The operation updates
// both persistent and live state.
//
// Returns error if:
//   - Node not found
//   - Already approved
//   - Storage update fails
func (n *nodesManager) ApproveNode(ctx context.Context, nodeID string) error {
	state, err := n.GetByPrefix(ctx, nodeID)
	if err != nil {
		return err
	}

	if state.Membership == models.NodeMembership.APPROVED {
		return NewErrNodeAlreadyApproved(nodeID)
	}

	state.Membership = models.NodeMembership.APPROVED
	return n.store.Put(ctx, state)
}

// RejectNode rejects a node from cluster participation.
// The operation:
//   - Updates node to REJECTED state
//   - Marks node as disconnected
//   - Removes live state tracking
//   - Triggers connection state change notification
//
// Returns error if:
//   - Node not found
//   - Already rejected
//   - Storage update fails
func (n *nodesManager) RejectNode(ctx context.Context, nodeID string) error {
	state, err := n.GetByPrefix(ctx, nodeID)
	if err != nil {
		return err
	}

	if state.Membership == models.NodeMembership.REJECTED {
		return NewErrNodeAlreadyRejected(nodeID)
	}

	// Update persistent state first
	state.Membership = models.NodeMembership.REJECTED
	state.ConnectionState.Status = models.NodeStates.DISCONNECTED
	state.ConnectionState.DisconnectedSince = n.clock.Now().UTC()
	state.ConnectionState.LastError = "node rejected"

	if err = n.store.Put(ctx, state); err != nil {
		return err
	}

	// Notify about connection state change if was connected
	if entry, exists := n.liveState.LoadAndDelete(state.Info.ID()); exists {
		if entry.(*trackedLiveState).connectionState.Status == models.NodeStates.CONNECTED {
			n.notifyConnectionStateChange(NodeConnectionEvent{
				NodeID:    state.Info.ID(),
				Previous:  models.NodeStates.CONNECTED,
				Current:   models.NodeStates.DISCONNECTED,
				Timestamp: n.clock.Now().UTC(),
			})
		}
	}

	return nil
}

// DeleteNode removes a node from the cluster.
// The operation:
//   - Removes node from persistent storage
//   - Removes live state tracking
//   - Triggers connection state change notification if was connected
//
// Returns error if:
//   - Node not found
//   - Storage deletion fails
func (n *nodesManager) DeleteNode(ctx context.Context, nodeID string) error {
	state, err := n.store.GetByPrefix(ctx, nodeID)
	if err != nil {
		return err
	}

	// Delete from persistent store first
	if err = n.store.Delete(ctx, state.Info.ID()); err != nil {
		return err
	}

	// Notify about connection state change if was connected
	if entry, exists := n.liveState.LoadAndDelete(state.Info.ID()); exists {
		if entry.(*trackedLiveState).connectionState.Status == models.NodeStates.CONNECTED {
			n.notifyConnectionStateChange(NodeConnectionEvent{
				NodeID:    state.Info.ID(),
				Previous:  models.NodeStates.CONNECTED,
				Current:   models.NodeStates.DISCONNECTED,
				Timestamp: n.clock.Now().UTC(),
			})
		}
	}

	return nil
}

// OnConnectionStateChange registers a handler for node connection state changes.
// Handlers are called synchronously when node state transitions between:
//   - CONNECTED <-> DISCONNECTED
//
// Events include:
//   - Previous and current state
//   - Timestamp of change
//   - Node identifier
func (n *nodesManager) OnConnectionStateChange(handler ConnectionStateChangeHandler) {
	n.handlers.Lock()
	defer n.handlers.Unlock()
	n.handlers.connectionState = append(n.handlers.connectionState, handler)
}

// notifyConnectionStateChange notifies all registered handlers of a state change
func (n *nodesManager) notifyConnectionStateChange(event NodeConnectionEvent) {
	// Update connected nodes counter
	if event.Current == models.NodeStates.CONNECTED {
		atomic.AddInt64(&n.connectedNodes, 1)
	} else if event.Previous == models.NodeStates.CONNECTED {
		atomic.AddInt64(&n.connectedNodes, -1)
	}

	n.handlers.RLock()
	defer n.handlers.RUnlock()

	for _, handler := range n.handlers.connectionState {
		handler(event)
	}
}

// isNodeDisconnected determines if a node should be considered disconnected
// based on its last heartbeat time and the configured disconnect timeout.
func (n *nodesManager) isNodeDisconnected(connState models.ConnectionState) bool {
	return connState.Status == models.NodeStates.CONNECTED &&
		n.clock.Since(connState.LastHeartbeat) > n.disconnectedAfter
}

//
// NodeReader interface implementation (keep existing methods)
//

func (n *nodesManager) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	state, err := n.store.Get(ctx, nodeID)
	if err != nil {
		return models.NodeState{}, err
	}
	n.enrichState(&state)
	return state, nil
}

func (n *nodesManager) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	state, err := n.store.GetByPrefix(ctx, prefix)
	if err != nil {
		return models.NodeState{}, err
	}
	n.enrichState(&state)
	return state, nil
}

func (n *nodesManager) List(ctx context.Context, filters ...NodeStateFilter) ([]models.NodeState, error) {
	states, err := n.store.List(ctx, filters...)
	if err != nil {
		return nil, err
	}

	for i := range states {
		n.enrichState(&states[i])
	}

	return states, nil
}

// resolveStartingOrchestratorSeqNum determines where a node should start receiving messages from.
//
// For reconnecting nodes, we trust the sequence numbers from our store rather than what the
// compute node reports. This prevents issues with compute nodes restarting with same ID but
// fresh state, where they would ask to start from 0.
//
// For new nodes, we start them from the latest sequence number to avoid overwhelming them
// with historical events.
//
// TODO: Add support for snapshots to allow nodes to catch up on missed state without
// replaying all historical events. For now, we always start from latest to avoid
// overwhelming nodes that have been down for a long time.
func (n *nodesManager) resolveStartingOrchestratorSeqNum(
	ctx context.Context, isReconnect bool, existing models.NodeState) (uint64, error) {
	if isReconnect {
		// For reconnecting nodes, trust our stored sequence number
		return existing.ConnectionState.LastOrchestratorSeqNum, nil
	}

	// For new nodes or nodes that have been gone too long,
	// start from latest to avoid overwhelming them
	latestSeq, err := n.eventstore.GetLatestEventNum(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest event number: %w", err)
	}

	return latestSeq, nil
}

// updateSequenceNumbers updates the last known sequence numbers for message tracking.
//   - LastOrchestratorSeqNum tracks what messages the compute node has processed
//   - LastComputeSeqNum tracks what messages the orchestrator has processed from this node. This is
//     populated locally by the orchestrator's data plane.
//
// TODO: Add smarter logic when updating sequence numbers by comparing current state versus observed states.
// Currently we trust what each node reports about their message processing:
//   - We trust what compute node says it has received from orchestrator (orchestratorSeqNum)
//   - We trust what orchestrator data plane says it processed from compute node (computeSeqNum)
//
// This simple approach could allow sequence numbers to move backwards in certain failure scenarios.
// We should implement proper comparison logic to ensure sequence numbers only advance forward.
func (n *nodesManager) updateSequenceNumbers(state *models.ConnectionState, orchestratorSeq, computeSeq uint64) {
	state.LastOrchestratorSeqNum = orchestratorSeq

	// Only update LastComputeSeqNum if greater than 0, as zero can indicate
	// either no messages processed yet or a connection that has just been
	// established. This preserves the existing sequence number in those cases.
	if computeSeq > 0 {
		state.LastComputeSeqNum = computeSeq
	}
}

// enrichState adds live tracking data to a node state.
// For connected nodes, it adds:
//   - Current connection state
//   - Available resource capacity
//   - Queue resource usage
//
// For disconnected nodes:
//   - Marks as disconnected
//   - Clears resource tracking
//   - Preserves disconnect timestamp
func (n *nodesManager) enrichState(state *models.NodeState) {
	// If we have live state, use it
	if entry, ok := n.liveState.Load(state.Info.ID()); ok {
		liveState := entry.(*trackedLiveState)
		state.ConnectionState = liveState.connectionState
		state.Info.ComputeNodeInfo.AvailableCapacity = liveState.availableCapacity
		state.Info.ComputeNodeInfo.QueueUsedCapacity = liveState.queueUsedCapacity
	} else {
		// If no live state exists, node is disconnected but keep the existing
		// ConnectionState (including DisconnectedSince) from persistent storage
		if state.Info.IsComputeNode() {
			state.ConnectionState.Status = models.NodeStates.DISCONNECTED
			// Clear resources since node is not connected
			state.Info.ComputeNodeInfo.AvailableCapacity = models.Resources{}
			state.Info.ComputeNodeInfo.QueueUsedCapacity = models.Resources{}
		}
	}
	//nolint:staticcheck
	state.Connection = state.ConnectionState.Status // for backward compatibility
}

func (n *nodesManager) selfRegister(ctx context.Context) error {
	// get latest node info
	nodeInfo := n.nodeInfoProvider.GetNodeInfo(ctx)

	// get node info from the store if it exists
	state, err := n.Get(ctx, nodeInfo.ID())
	if err != nil {
		if !bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError) {
			return bacerrors.Newf("failed to self-register node: %v", err).
				WithComponent(errComponent)
		}
		state = models.NodeState{
			Info:       nodeInfo,
			Membership: models.NodeMembership.APPROVED,
			ConnectionState: models.ConnectionState{
				ConnectedSince: n.clock.Now().UTC(),
			},
		}
	}
	// update the node info and make as connected
	state.Info = nodeInfo
	state.ConnectionState.Status = models.NodeStates.CONNECTED
	state.ConnectionState.LastHeartbeat = n.clock.Now().UTC()

	// for backward compatibility before connection state was introduced
	if state.ConnectionState.ConnectedSince.IsZero() {
		state.ConnectionState.ConnectedSince = n.clock.Now().UTC()
	}

	// store the updated state
	if err = n.store.Put(ctx, state); err != nil {
		return bacerrors.Newf("failed to self-register node: %v", err).
			WithComponent(errComponent)
	}

	return nil
}

// GetConnectedNodesCount returns the number of currently connected nodes.
func (n *nodesManager) GetConnectedNodesCount() int {
	return int(atomic.LoadInt64(&n.connectedNodes))
}

// compile-time check that nodesManager implements the Manager interface
var _ Manager = &nodesManager{}
