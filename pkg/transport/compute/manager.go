package compute

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
)

// ConnectionManager handles the lifecycle and maintenance of a compute node's connection
// to the orchestrator. It manages connection state, health monitoring, heartbeats,
// and node information updates.
type ConnectionManager struct {
	config   Config
	natsConn *nats.Conn
	state    atomic.Int32 // Atomic state tracking (Connected, Disconnected, etc)

	// Core components
	nodeInfoProvider models.NodeInfoProvider // Provides node information updates
	requester        ncl.Requester           // Handles control plane communication
	subscriber       ncl.Subscriber          // Handles incoming data plane messages
	dataPlane        *DataPlane              // Handles data plane operations

	// Sequence trackers
	incomingCheckpointName string
	incomingSeqTracker     *core.SequenceTracker

	// Health tracking information
	health struct {
		sync.RWMutex
		lastHeartbeat      time.Time // Time of last successful heartbeat
		lastUpdate         time.Time // Time of last successful node info update
		connectedSince     time.Time // When the current connection was established
		consecutiveErrors  int       // Count of consecutive connection failures
		lastError          error     // Most recent error encountered
		lastConnectAttempt time.Time // Time of last connection attempt
	}

	// State tracking
	latestNodeInfo models.NodeInfo // Most recent node information
	backoff        backoff.Backoff // Handles backoff between reconnection attempts
	startTime      time.Time       // When the connection manager was started

	// Control channels and synchronization
	stopCh chan struct{}  // Channel for signaling shutdown
	wg     sync.WaitGroup // WaitGroup for background goroutines

	// Event handlers
	stateHandlers []core.ConnectionStateHandler // Callbacks for connection state changes
	mu            sync.RWMutex                  // Protects state handlers and critical sections
}

// NewConnectionManager creates a new connection manager with the given configuration.
// It initializes the manager but does not start any connections - Start() must be called.
func NewConnectionManager(cfg Config) (*ConnectionManager, error) {
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cm := &ConnectionManager{
		config:                 cfg,
		nodeInfoProvider:       cfg.NodeInfoProvider,
		incomingCheckpointName: fmt.Sprintf("incoming-%s", cfg.NodeID),
		stopCh:                 make(chan struct{}),
		backoff:                cfg.Backoff,
		startTime:              time.Now(),
	}

	return cm, nil
}

// Start begins the connection management process. It launches background goroutines for:
// - Connection maintenance
// - Heartbeat sending
// - Node info updates
func (cm *ConnectionManager) Start(ctx context.Context) error {
	log.Info().
		Str("node_id", cm.config.NodeID).
		Time("start_time", cm.startTime).
		Msg("Starting connection manager")

	cm.latestNodeInfo = cm.config.NodeInfoProvider.GetNodeInfo(ctx)

	// initialize sequence tracker
	checkpoint, err := cm.config.Checkpointer.GetCheckpoint(ctx, cm.incomingCheckpointName)
	if err != nil {
		return fmt.Errorf("failed to get last checkpoint: %w", err)
	}
	cm.incomingSeqTracker = core.NewSequenceTracker().WithLastSeqNum(checkpoint)

	// Start connection management in background
	cm.wg.Add(1)
	go cm.maintainConnection(ctx)

	// Start heartbeat in background
	cm.wg.Add(1)
	go cm.maintainHeartbeat(ctx)

	// Start node info updates in background
	cm.wg.Add(1)
	go cm.maintainNodeInfoUpdates(ctx)

	// Start checkpoint updates in background
	cm.wg.Add(1)
	go cm.maintainProgressCheckpoints(ctx)

	return nil
}

// Close gracefully shuts down the connection manager and all its components.
// It waits for background goroutines to complete or until the context is cancelled.
func (cm *ConnectionManager) Close(ctx context.Context) error {
	close(cm.stopCh)
	defer cm.cleanup(ctx)

	// Wait with timeout for graceful shutdown
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// cleanup performs orderly cleanup of connection manager components:
// 1. Stops the data plane
// 2. Cleans up control plane
// 3. Closes NATS connection
func (cm *ConnectionManager) cleanup(ctx context.Context) {
	// Clean up data plane subscriber
	if cm.subscriber != nil {
		if err := cm.subscriber.Close(context.Background()); err != nil {
			log.Error().Err(err).Msg("Failed to close subscriber")
		}
		cm.subscriber = nil
	}

	// Clean up data plane
	if cm.dataPlane != nil {
		if err := cm.dataPlane.Stop(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to stop data plane")
		}
		cm.dataPlane = nil
	}

	// Clean up control plane
	cm.requester = nil

	// Clean up NATS connection last
	if cm.natsConn != nil {
		cm.natsConn.Close()
		cm.natsConn = nil
	}
}

// maintainConnection runs a periodic loop that:
// - Attempts to connect when disconnected
// - Checks connection health when connected
// - Handles reconnection with backoff on failures
func (cm *ConnectionManager) maintainConnection(ctx context.Context) {
	defer cm.wg.Done()

	// Attempt initial connection immediately
	if cm.GetState() == core.Disconnected {
		if err := cm.connect(ctx); err != nil {
			backoffDuration := cm.backoff.BackoffDuration(cm.health.consecutiveErrors)
			log.Error().
				Err(err).
				Int("consecutive_errors", cm.health.consecutiveErrors).
				Dur("backoff_duration", backoffDuration).
				Msg("Initial connection attempt failed")
			cm.backoff.Backoff(ctx, cm.health.consecutiveErrors)
		}
	}

	ticker := time.NewTicker(cm.config.ReconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			currentState := cm.GetState()

			// Check if we need to connect
			if currentState == core.Disconnected {
				if err := cm.connect(ctx); err != nil {
					backoffDuration := cm.backoff.BackoffDuration(cm.health.consecutiveErrors)
					log.Error().
						Err(err).
						Int("consecutive_errors", cm.health.consecutiveErrors).
						Dur("backoff_duration", backoffDuration).
						Msg("Connection attempt failed")
					cm.backoff.Backoff(ctx, cm.health.consecutiveErrors)
					continue
				}
				continue
			}

			// Check connection health if we're connected
			if currentState == core.Connected {
				healthy := cm.checkConnectionHealth()
				if !healthy {
					log.Warn().Msg("Connection unhealthy, initiating reconnect")
					cm.mu.Lock()
					cm.transitionState(core.Disconnected, fmt.Errorf("connection unhealthy"))
					cm.mu.Unlock()
					continue
				}
			}
		}
	}
}

// checkConnectionHealth verifies the connection is healthy by checking:
// - Recent heartbeat activity
// - NATS connection status
func (cm *ConnectionManager) checkConnectionHealth() bool {
	cm.health.RLock()
	defer cm.health.RUnlock()

	// Consider connection unhealthy if:
	// 1. No heartbeat received within 3 intervals
	// 2. NATS connection is closed/draining
	now := time.Now()
	heartbeatDeadline := now.Add(-3 * cm.config.HeartbeatInterval)

	isHealthy := true
	if cm.health.lastHeartbeat.Before(heartbeatDeadline) {
		log.Warn().
			Time("last_heartbeat", cm.health.lastHeartbeat).
			Time("deadline", heartbeatDeadline).
			Msg("No recent heartbeat published")
		isHealthy = false
	}

	if cm.natsConn.IsClosed() {
		log.Warn().Msg("NATS connection is closed")
		isHealthy = false
	}

	return isHealthy
}

// GetHealth returns the current health status of the connection including:
// - Timestamps of last successful operations
// - Current connection state
// - Error counts and details
func (cm *ConnectionManager) GetHealth() core.ConnectionHealth {
	cm.health.RLock()
	defer cm.health.RUnlock()

	return core.ConnectionHealth{
		LastSuccessfulHeartbeat: cm.health.lastHeartbeat,
		LastSuccessfulUpdate:    cm.health.lastUpdate,
		CurrentState:            cm.GetState(),
		ConsecutiveFailures:     cm.health.consecutiveErrors,
		LastError:               cm.health.lastError,
		ConnectedSince:          cm.health.connectedSince,
	}
}

// transitionState handles state transitions between Connected/Disconnected/Connecting.
// It updates health metrics and notifies registered state change handlers.
func (cm *ConnectionManager) transitionState(newState core.ConnectionState, err error) {
	oldState := core.ConnectionState(cm.state.Load())
	if oldState == newState {
		return
	}

	// Update state
	cm.state.Store(int32(newState))

	// Update health tracking
	cm.health.Lock()
	if newState == core.Connected {
		cm.health.connectedSince = time.Now()
		cm.health.lastHeartbeat = time.Now() // Reset heartbeat on connection
		cm.health.consecutiveErrors = 0
		cm.health.lastError = nil
	} else if newState == core.Disconnected {
		cm.health.lastError = err
		cm.health.consecutiveErrors++
	}
	cm.health.Unlock()

	// Notify handlers
	for _, handler := range cm.stateHandlers {
		handler(newState)
	}

	log.Debug().
		Str("old_state", oldState.String()).
		Str("new_state", newState.String()).
		Err(err).
		Msg("Connection state changed")
}

// connect establishes a new connection to the orchestrator:
// 1. Creates NATS connection
// 2. Sets up control plane
// 3. Performs handshake
// 4. Initializes data plane
func (cm *ConnectionManager) connect(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Info().
		Str("node_id", cm.config.NodeID).
		Msg("Attempting to establish connection")

	cm.transitionState(core.Connecting, nil)

	var err error
	defer func() {
		if err != nil {
			cm.cleanup(ctx)
			cm.transitionState(core.Disconnected, err)
		}
	}()

	// Cleanup any existing connections
	cm.cleanup(ctx)

	// Create NATS connection
	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create control plane requester
	cm.requester, err = ncl.NewRequester(cm.natsConn, ncl.RequesterConfig{
		Name:              cm.config.NodeID,
		Destination:       core.NatsSubjectComputeOutCtrl(cm.config.NodeID),
		MessageSerializer: cm.config.MessageSerializer,
		MessageRegistry:   cm.config.MessageRegistry,
	})
	if err != nil {
		return fmt.Errorf("failed to create requester: %w", err)
	}

	// Create data plane message handler before the handshake to avoid missing messages
	cm.subscriber, err = ncl.NewSubscriber(cm.natsConn, ncl.SubscriberConfig{
		Name:               cm.config.NodeID,
		MessageRegistry:    cm.config.MessageRegistry,
		MessageSerializer:  cm.config.MessageSerializer,
		MessageHandler:     cm.config.DataPlaneMessageHandler,
		ProcessingNotifier: cm.incomingSeqTracker,
	})

	// Subscribe to incoming messages
	if err = cm.subscriber.Subscribe(ctx, core.NatsSubjectComputeInMsgs(cm.config.NodeID)); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Perform handshake
	var handshakeResponse messages.HandshakeResponse
	handshakeResponse, err = cm.performHandshake(ctx, cm.latestNodeInfo)
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}
	if !handshakeResponse.Accepted {
		return fmt.Errorf("handshake rejected: %s", handshakeResponse.Reason)
	}

	// Create data plane only after successful handshake
	cm.dataPlane, err = NewDataPlane(DataPlaneConfig{
		NodeID:             cm.config.NodeID,
		Client:             cm.natsConn,
		MessageRegistry:    cm.config.MessageRegistry,
		MessageSerializer:  cm.config.MessageSerializer,
		MessageCreator:     cm.config.DataPlaneMessageCreator,
		EventStore:         cm.config.EventStore,
		DispatcherConfig:   cm.config.DispatcherConfig,
		LogStreamServer:    cm.config.LogStreamServer,
		LastReceivedSeqNum: handshakeResponse.LastComputeSeqNum,
	})
	if err != nil {
		return fmt.Errorf("failed to create data plane: %w", err)
	}

	// Start data plane
	if err = cm.dataPlane.Start(ctx); err != nil {
		return fmt.Errorf("failed to start data plane: %w", err)
	}

	cm.transitionState(core.Connected, nil)
	return nil
}

// performHandshake executes the initial handshake with the orchestrator
// sending node information and start time
func (cm *ConnectionManager) performHandshake(
	ctx context.Context, info models.NodeInfo) (messages.HandshakeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, cm.config.RequestTimeout)
	defer cancel()

	handshake := messages.HandshakeRequest{
		NodeInfo:               info,
		StartTime:              cm.startTime,
		LastOrchestratorSeqNum: cm.incomingSeqTracker.GetLastSeqNum(),
	}

	// Send handshake
	msg := envelope.NewMessage(handshake).
		WithMetadataValue(envelope.KeyMessageType, messages.HandshakeRequestMessageType)

	requester := cm.requester
	if requester == nil {
		// TODO: fix potential race condition
		return messages.HandshakeResponse{}, fmt.Errorf("requester has been closed")
	}
	response, err := requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return messages.HandshakeResponse{}, fmt.Errorf("handshake request failed: %w", err)
	}

	payload, ok := response.GetPayload(messages.HandshakeResponse{})
	if !ok {
		return messages.HandshakeResponse{}, fmt.Errorf(
			"invalid handshake response payload. expected messages.HandshakeResponse, got %T", payload)
	}

	return payload.(messages.HandshakeResponse), nil
}

// maintainHeartbeat runs a periodic loop sending heartbeats when connected
func (cm *ConnectionManager) maintainHeartbeat(ctx context.Context) {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			if cm.GetState() == core.Connected {
				if err := cm.sendHeartbeat(ctx); err != nil {
					log.Error().Err(err).Msg("failed to send heartbeat")
					continue
				}
			}
		}
	}
}

// sendHeartbeat sends a heartbeat message to the orchestrator with current capacity information
func (cm *ConnectionManager) sendHeartbeat(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, cm.config.RequestTimeout)
	defer cancel()

	nodeInfo := cm.nodeInfoProvider.GetNodeInfo(ctx)
	cm.latestNodeInfo = nodeInfo
	var availableCapacity models.Resources
	var queueUsedCapacity models.Resources
	if nodeInfo.ComputeNodeInfo != nil {
		availableCapacity = nodeInfo.ComputeNodeInfo.AvailableCapacity
		queueUsedCapacity = nodeInfo.ComputeNodeInfo.QueueUsedCapacity
	}

	msg := envelope.NewMessage(messages.HeartbeatRequest{
		NodeID:                 cm.latestNodeInfo.NodeID,
		AvailableCapacity:      availableCapacity,
		QueueUsedCapacity:      queueUsedCapacity,
		LastOrchestratorSeqNum: cm.incomingSeqTracker.GetLastSeqNum(),
	}).WithMetadataValue(envelope.KeyMessageType, messages.HeartbeatRequestMessageType)

	requester := cm.requester
	if requester == nil {
		// TODO: fix potential race condition
		return fmt.Errorf("requester has been closed")
	}

	response, err := cm.requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return fmt.Errorf("heartbeat request failed: %w", err)
	}

	payload, ok := response.GetPayload(messages.HeartbeatResponse{})
	if !ok {
		return fmt.Errorf("invalid heartbeat response payload. expected messages.HeartbeatResponse, got %T", payload)
	}

	// Update health on successful heartbeat
	cm.health.Lock()
	cm.health.lastHeartbeat = time.Now()
	cm.health.Unlock()

	return nil
}

// maintainNodeInfoUpdates runs a periodic loop checking for node info changes
// and sending updates when changes are detected
func (cm *ConnectionManager) maintainNodeInfoUpdates(ctx context.Context) {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.NodeInfoUpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			if cm.GetState() == core.Connected {
				prevNodeInfo := cm.latestNodeInfo
				cm.latestNodeInfo = cm.nodeInfoProvider.GetNodeInfo(ctx)

				if models.HasNodeInfoChanged(prevNodeInfo, cm.latestNodeInfo) {
					log.Debug().Msg("Node info changed, sending update")
					if err := cm.sendNodeInfoUpdate(ctx, cm.latestNodeInfo); err != nil {
						// Log error but continue
						log.Error().Err(err).Msg("failed to send node info update")
						continue
					}
				}
			}
		}
	}
}

// sendNodeInfoUpdate sends updated node information to the orchestrator.
// It includes the latest node state, capacity, and configuration changes.
// Updates health tracking on successful updates.
func (cm *ConnectionManager) sendNodeInfoUpdate(ctx context.Context, nodeInfo models.NodeInfo) error {
	ctx, cancel := context.WithTimeout(ctx, cm.config.RequestTimeout)
	defer cancel()

	msg := envelope.NewMessage(messages.UpdateNodeInfoRequest{
		NodeInfo: nodeInfo,
	}).WithMetadataValue(envelope.KeyMessageType, messages.NodeInfoUpdateRequestMessageType)

	_, err := cm.requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return fmt.Errorf("node info update request failed: %w", err)
	}

	// Update health tracking with successful update time
	cm.health.Lock()
	cm.health.lastUpdate = time.Now()
	cm.health.Unlock()

	return nil
}

// maintainProgressCheckpoints runs a periodic loop to checkpoint the last received sequence number
func (cm *ConnectionManager) maintainProgressCheckpoints(ctx context.Context) {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.CheckpointInterval)
	defer ticker.Stop()

	lastCheckpoint := cm.incomingSeqTracker.GetLastSeqNum()
	doCheckpoint := func() {
		newCheckpoint := cm.incomingSeqTracker.GetLastSeqNum()
		if newCheckpoint == lastCheckpoint {
			return
		}
		if err := cm.config.Checkpointer.Checkpoint(ctx, cm.incomingCheckpointName, newCheckpoint); err != nil {
			log.Error().Err(err).Msg("failed to checkpoint incoming sequence number")
		} else {
			lastCheckpoint = newCheckpoint
		}

	}

	for {
		select {
		case <-cm.stopCh:
			doCheckpoint()
			return
		case <-ticker.C:
			doCheckpoint()
		}
	}
}

// setState atomically updates the connection state and notifies handlers
// if the state has changed. This is the low-level state change mechanism
// used by transitionState.
func (cm *ConnectionManager) setState(state core.ConnectionState) {
	old := core.ConnectionState(cm.state.Swap(int32(state)))
	if old != state {
		cm.notifyStateChange(state)
	}
}

// GetState returns the current connection state.
// Uses atomic operations to safely read the state value.
func (cm *ConnectionManager) GetState() core.ConnectionState {
	return core.ConnectionState(cm.state.Load())
}

// notifyStateChange calls all registered state change handlers with the new state.
// Handlers are called while holding a read lock to prevent modification
// of the handler list during notification.
func (cm *ConnectionManager) notifyStateChange(state core.ConnectionState) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	for _, handler := range cm.stateHandlers {
		handler(state)
	}
}

// OnStateChange registers a new handler to be called when the connection
// state changes. Handlers are called synchronously when state transitions occur.
func (cm *ConnectionManager) OnStateChange(handler core.ConnectionStateHandler) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.stateHandlers = append(cm.stateHandlers, handler)
}
