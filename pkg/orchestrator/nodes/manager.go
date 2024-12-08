package nodes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

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
)

// nodesManager handles node lifecycle, health checking, and state management.
// It tracks node connection states, handles handshakes and heartbeats, and
// maintains node membership status.
type nodesManager struct {
	// Core dependencies
	store Store       // Persistent storage for node states
	clock clock.Clock // Time source (can be mocked for testing)

	// Configuration
	defaultApprovalState    models.NodeMembershipState // Initial membership state for new nodes
	heartbeatCheckFrequency time.Duration              // How often to check node health
	disconnectedAfter       time.Duration              // Time after which to mark nodes as disconnected
	persistInterval         time.Duration              // For periodic persistence
	persistTimeout          time.Duration
	shutdownTimeout         time.Duration

	// Runtime state
	liveState *sync.Map // Thread-safe map of nodeID -> trackedLiveState

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

// NodeManagerParams holds the configuration for creating a new nodesManager
type NodeManagerParams struct {
	Store                 Store         // Required: persistent storage
	Clock                 clock.Clock   // Optional: defaults to real clock
	NodeDisconnectedAfter time.Duration // Required: timeout for node health
	HealthCheckFrequency  time.Duration // Optional: defaults to 1/3 of disconnectedAfter
	ManualApproval        bool          // Whether nodes need manual approval
	PersistInterval       time.Duration // Interval for state persistence
	PersistTimeout        time.Duration // Timeout for state persistence
	ShutdownTimeout       time.Duration // Timeout for graceful shutdown
}

// trackedLiveState holds the current resource state for a node
type trackedLiveState struct {
	connectionState   models.ConnectionState
	availableCapacity models.Resources
	queueUsedCapacity models.Resources
}

// NewNodeManager creates a new nodesManager with the given configuration.
// It initializes the manager but does not start background tasks - call Start() for that.
func NewNodeManager(params NodeManagerParams) (Manager, error) {
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

	return &nodesManager{
		store:                   params.Store,
		clock:                   params.Clock,
		liveState:               &sync.Map{},
		defaultApprovalState:    defaultApprovalState,
		heartbeatCheckFrequency: heartbeatCheckFrequency,
		disconnectedAfter:       params.NodeDisconnectedAfter,
		persistInterval:         params.PersistInterval,
		persistTimeout:          params.PersistTimeout,
		shutdownTimeout:         params.ShutdownTimeout,
		stopCh:                  make(chan struct{}),
	}, nil
}

// Start initializes the nodesManager and begins background tasks.
// It loads existing node states from storage and starts health checking.
// Returns an error if the manager is already running or if state loading fails.
func (n *nodesManager) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		return errors.New("node manager already running")
	}

	// Initialize clean state
	n.liveState = &sync.Map{}
	n.stopCh = make(chan struct{})

	// Start background tasks
	n.startBackgroundTask("health-check", n.healthCheckLoop)
	n.startBackgroundTask("state-persistence", n.persistenceLoop)

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

// Stop gracefully shuts down the nodesManager and its background tasks.
// It waits for tasks to complete or until the context is cancelled.
// Returns nil if already stopped or successfully stopped, context error if timed out.
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

// Running returns true if the nodesManager is currently running.
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
// It marks nodes as disconnected if they haven't sent a heartbeat within the timeout period.
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
		newConnectionState.DisconnectedSince = n.clock.Now()
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
			Timestamp: n.clock.Now(),
		})
	}
}

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

func (n *nodesManager) Handshake(
	ctx context.Context, request messages.HandshakeRequest) (messages.HandshakeResponse, error) {
	log.Debug().Msgf("handshake request received with info %+v", request.NodeInfo)

	existingConnectionState := models.NodeStates.DISCONNECTED
	isReconnect := false
	var existing models.NodeState

	// Check if node is already registered, and if so, if it was rejected
	existing, err := n.store.Get(ctx, request.NodeInfo.ID())
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
			Status:                 models.NodeStates.CONNECTED,
			ConnectedSince:         n.clock.Now(),
			LastHeartbeat:          n.clock.Now(),
			LastOrchestratorSeqNum: request.LastOrchestratorSeqNum,
		},
	}

	if isReconnect {
		state.Membership = existing.Membership
		state.ConnectionState.LastComputeSeqNum = existing.ConnectionState.LastComputeSeqNum
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
		Timestamp: n.clock.Now(),
	})

	log.Info().Msgf("handshake successful with node %s", request.NodeInfo.ID())

	reason := "node registered"
	if isReconnect {
		reason = "node reconnected"
	}
	return messages.HandshakeResponse{
		Accepted:          true,
		Reason:            reason,
		LastComputeSeqNum: state.ConnectionState.LastComputeSeqNum,
	}, nil
}

func (n *nodesManager) UpdateNodeInfo(
	ctx context.Context, request messages.UpdateNodeInfoRequest) (messages.UpdateNodeInfoResponse, error) {
	existing, err := n.Get(ctx, request.NodeInfo.ID())
	if err != nil {
		if errors.As(err, &ErrNodeNotFound{}) {
			// return an error that a handshake is first required
			return messages.UpdateNodeInfoResponse{},
				fmt.Errorf("node %s not yet registered - handshake required", request.NodeInfo.ID())
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

func (n *nodesManager) Heartbeat(
	ctx context.Context, request ExtendedHeartbeatRequest) (messages.HeartbeatResponse, error) {
	// Retry loop for concurrent updates, such as handshakes or health checks
	maxConcurrentAttempts := 3
	for i := 0; i < maxConcurrentAttempts; i++ {
		// Get existing live state
		existingEntry, ok := n.liveState.Load(request.NodeID)
		if !ok {
			return messages.HeartbeatResponse{}, fmt.Errorf("node %s not connected - handshake required", request.NodeID)
		}

		existing := existingEntry.(*trackedLiveState)
		if existing.connectionState.Status != models.NodeStates.CONNECTED {
			return messages.HeartbeatResponse{}, fmt.Errorf("node %s not connected - handshake required", request.NodeID)
		}

		// updated connection state
		updated := existing.connectionState
		updated.LastHeartbeat = n.clock.Now()
		if request.LastOrchestratorSeqNum > 0 {
			updated.LastOrchestratorSeqNum = request.LastOrchestratorSeqNum
		}
		if request.LastComputeSeqNum > 0 {
			updated.LastComputeSeqNum = request.LastComputeSeqNum
		}

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
	return messages.HeartbeatResponse{}, errors.New("concurrent update conflict")
}

func (n *nodesManager) ApproveNode(ctx context.Context, nodeID string) error {
	state, err := n.GetByPrefix(ctx, nodeID)
	if err != nil {
		return err
	}

	if state.Membership == models.NodeMembership.APPROVED {
		return errors.New("node already approved")
	}

	state.Membership = models.NodeMembership.APPROVED
	return n.store.Put(ctx, state)
}

func (n *nodesManager) RejectNode(ctx context.Context, nodeID string) error {
	state, err := n.GetByPrefix(ctx, nodeID)
	if err != nil {
		return err
	}

	if state.Membership == models.NodeMembership.REJECTED {
		return errors.New("node already rejected")
	}

	// Update persistent state first
	state.Membership = models.NodeMembership.REJECTED
	state.ConnectionState.Status = models.NodeStates.DISCONNECTED
	state.ConnectionState.DisconnectedSince = n.clock.Now()
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
				Timestamp: n.clock.Now(),
			})
		}
	}

	return nil
}

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
				Timestamp: n.clock.Now(),
			})
		}
	}

	return nil
}

func (n *nodesManager) OnConnectionStateChange(handler ConnectionStateChangeHandler) {
	n.handlers.Lock()
	defer n.handlers.Unlock()
	n.handlers.connectionState = append(n.handlers.connectionState, handler)
}

// notifyConnectionStateChange notifies all registered handlers of a state change
func (n *nodesManager) notifyConnectionStateChange(event NodeConnectionEvent) {
	n.handlers.RLock()
	defer n.handlers.RUnlock()

	for _, handler := range n.handlers.connectionState {
		handler(event)
	}
}

// isNodeDisconnected checks if a node is considered disconnected based on its connection state
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

// enrichState adds both connection state and resources to the node state
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

// compile-time check that nodesManager implements the Manager interface
var _ Manager = &nodesManager{}
