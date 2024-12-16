package compute

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

const stateChangesBuffer = 32

// ConnectionManager handles the lifecycle of a compute node's connection to the orchestrator.
// It manages the complete connection lifecycle including:
//   - Initial connection and handshake
//   - Connection health monitoring
//   - Automated reconnection with backoff
//   - Control and data plane management
//   - Connection state transitions
type ConnectionManager struct {
	// Configuration for the connection manager
	config Config

	// Active NATS connection
	natsConn *nats.Conn

	// Core messaging components
	subscriber   ncl.Subscriber // Handles incoming data plane messages
	controlPlane *ControlPlane  // Manages periodic operations when connected
	dataPlane    *DataPlane     // Handles outgoing message dispatch

	// Checkpointing configuration
	incomingCheckpointName string                       // Name used for checkpoint storage
	incomingSeqTracker     *nclprotocol.SequenceTracker // Tracks processed message sequences

	// Health monitoring
	healthTracker *HealthTracker // Tracks connection health and state

	// Lifecycle management
	running bool           // Whether the manager is currently running
	stopCh  chan struct{}  // Signals shutdown to background goroutines
	wg      sync.WaitGroup // Tracks active background goroutines

	// Event handling
	stateHandlers   []nclprotocol.ConnectionStateHandler // Callbacks for state transitions
	stateHandlersMu sync.RWMutex
	stateChanges    chan stateChange // Channel for ordered state change notifications
	mu              sync.RWMutex     // Protects shared state access
}

type stateChange struct {
	state nclprotocol.ConnectionState
	err   error
}

// NewConnectionManager creates a new connection manager with the given configuration.
// It initializes the manager but does not start any connections - Start() must be called.
func NewConnectionManager(cfg Config) (*ConnectionManager, error) {
	cfg.SetDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cm := &ConnectionManager{
		config:                 cfg,
		healthTracker:          NewHealthTracker(cfg.Clock),
		incomingCheckpointName: fmt.Sprintf("incoming-%s", cfg.NodeID),
		stopCh:                 make(chan struct{}),
		stateChanges:           make(chan stateChange, stateChangesBuffer), // buffered to avoid blocking
	}

	return cm, nil
}

// Start begins the connection management process. It launches background goroutines for:
// - Connection maintenance
// - Heartbeat sending
// - Node info updates
func (cm *ConnectionManager) Start(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.running {
		return bacerrors.New("connection manager already running").
			WithCode(bacerrors.BadRequestError).
			WithComponent(errComponent)
	}

	log.Info().
		Str("node_id", cm.config.NodeID).
		Time("start_time", cm.healthTracker.GetHealth().StartTime).
		Msg("Starting connection manager")

	// initialize sequence tracker
	checkpoint, err := cm.config.Checkpointer.GetCheckpoint(ctx, cm.incomingCheckpointName)
	if err != nil {
		return fmt.Errorf("failed to get last checkpoint: %w", err)
	}
	cm.incomingSeqTracker = nclprotocol.NewSequenceTracker().WithLastSeqNum(checkpoint)

	// create new channels in case the connection manager is restarted
	cm.stopCh = make(chan struct{})
	cm.stateChanges = make(chan stateChange, stateChangesBuffer)

	// Start connection management in background
	cm.wg.Add(1)
	go cm.maintainConnectionLoop(context.TODO())

	// Start state change notification handler
	cm.wg.Add(1)
	go cm.handleStateChanges()

	cm.running = true
	return nil
}

// Close gracefully shuts down the connection manager and all its components.
// It waits for background goroutines to complete or until the context is cancelled.
func (cm *ConnectionManager) Close(ctx context.Context) error {
	cm.mu.Lock()
	if !cm.running {
		cm.mu.Unlock()
		return nil
	}
	cm.running = false
	close(cm.stopCh)
	cm.mu.Unlock()

	// Wait for graceful shutdown
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(cm.stateChanges)
		close(done)
	}()

	select {
	case <-done:
		return cm.cleanup(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// cleanup performs orderly cleanup of connection manager components:
// 1. Stops the data plane
// 2. Cleans up control plane
// 3. Closes NATS connection
func (cm *ConnectionManager) cleanup(ctx context.Context) error {
	var errs error
	// Clean up data plane subscriber
	if cm.subscriber != nil {
		if err := cm.subscriber.Close(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to close subscriber: %w", err))
		}
		cm.subscriber = nil
	}

	// Clean up data plane
	if cm.dataPlane != nil {
		if err := cm.dataPlane.Stop(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop data plane: %w", err))
		}
		cm.dataPlane = nil
	}

	// Clean up control plane
	if cm.controlPlane != nil {
		if err := cm.controlPlane.Stop(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to stop control plane: %w", err))
		}
		cm.controlPlane = nil
	}

	// Clean up NATS connection last
	if cm.natsConn != nil {
		cm.natsConn.Close()
		cm.natsConn = nil
	}

	return errs
}

// connect attempts to establish a connection to the orchestrator. It follows these steps:
// 1. Creates NATS connection and transport components
// 2. Performs initial handshake with orchestrator
// 3. Sets up and starts control and data planes
func (cm *ConnectionManager) connect(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.getState() == nclprotocol.Connected {
		return nil
	}

	log.Info().Str("node_id", cm.config.NodeID).Msg("Attempting to establish connection")
	cm.transitionState(nclprotocol.Connecting, nil)

	// cleanup existing components before reconnecting
	if err := cm.cleanup(ctx); err != nil {
		return fmt.Errorf("failed to cleanup existing components: %w", err)
	}

	var err error
	defer func() {
		if err != nil {
			if cleanupErr := cm.cleanup(ctx); cleanupErr != nil {
				log.Warn().Err(cleanupErr).Msg("failed to cleanup after connection error")
			}
			cm.transitionState(nclprotocol.Disconnected, err)
		}
	}()

	if err = cm.setupTransport(ctx); err != nil {
		return fmt.Errorf("failed to setup transport: %w", err)
	}

	if err = cm.setupSubscriber(ctx); err != nil {
		return fmt.Errorf("failed to setup subscriber: %w", err)
	}

	requester, err := cm.setupRequester(ctx)
	if err != nil {
		return fmt.Errorf("failed to setup requester: %w", err)
	}

	handshakeResponse, err := cm.performHandshake(ctx, requester)
	if err != nil {
		return fmt.Errorf("handshake failed: %w", err)
	}

	if err = cm.setupControlPlane(ctx, requester); err != nil {
		return fmt.Errorf("failed to setup control plane: %w", err)
	}

	if err = cm.setupDataPlane(ctx, handshakeResponse); err != nil {
		return fmt.Errorf("failed to setup data plane: %w", err)
	}

	cm.transitionState(nclprotocol.Connected, nil)
	return nil
}

// setupTransport creates the NATS connection
func (cm *ConnectionManager) setupTransport(ctx context.Context) error {
	var err error
	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return nil
}

// setupRequester creates the control plane publisher
func (cm *ConnectionManager) setupRequester(ctx context.Context) (ncl.Publisher, error) {
	return ncl.NewPublisher(cm.natsConn, ncl.PublisherConfig{
		Name:              cm.config.NodeID,
		Destination:       nclprotocol.NatsSubjectComputeOutCtrl(cm.config.NodeID),
		MessageSerializer: cm.config.MessageSerializer,
		MessageRegistry:   cm.config.MessageRegistry,
	})
}

// setupSubscriber creates and starts the data plane message subscriber
func (cm *ConnectionManager) setupSubscriber(ctx context.Context) error {
	var err error
	cm.subscriber, err = ncl.NewSubscriber(cm.natsConn, ncl.SubscriberConfig{
		Name:               cm.config.NodeID,
		MessageRegistry:    cm.config.MessageRegistry,
		MessageSerializer:  cm.config.MessageSerializer,
		MessageHandler:     cm.config.DataPlaneMessageHandler,
		ProcessingNotifier: cm.incomingSeqTracker,
	})
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	if err = cm.subscriber.Subscribe(ctx, nclprotocol.NatsSubjectComputeInMsgs(cm.config.NodeID)); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	return nil
}

// performHandshake executes the initial handshake with the orchestrator
// sending node information and start time
func (cm *ConnectionManager) performHandshake(
	ctx context.Context, requester ncl.Publisher) (messages.HandshakeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, cm.config.RequestTimeout)
	defer cancel()

	handshake := messages.HandshakeRequest{
		NodeInfo:               cm.config.NodeInfoProvider.GetNodeInfo(ctx),
		StartTime:              cm.GetHealth().StartTime,
		LastOrchestratorSeqNum: cm.incomingSeqTracker.GetLastSeqNum(),
	}

	// Send handshake
	msg := envelope.NewMessage(handshake).
		WithMetadataValue(envelope.KeyMessageType, messages.HandshakeRequestMessageType)

	response, err := requester.Request(ctx, ncl.NewPublishRequest(msg))
	if err != nil {
		return messages.HandshakeResponse{}, fmt.Errorf("handshake request failed: %w", err)
	}

	payload, ok := response.GetPayload(messages.HandshakeResponse{})
	if !ok {
		return messages.HandshakeResponse{}, fmt.Errorf(
			"invalid handshake response payload. expected messages.HandshakeResponse, got %T", payload)
	}

	handshakeResponse := payload.(messages.HandshakeResponse)
	if !handshakeResponse.Accepted {
		return messages.HandshakeResponse{}, fmt.Errorf(
			"handshake rejected by orchestrator due to %s", handshakeResponse.Reason)
	}

	// Always trust the orchestrator's starting sequence number as it may have been reset
	// or decided to start from a different point
	cm.incomingSeqTracker.UpdateLastSeqNum(handshakeResponse.StartingOrchestratorSeqNum)

	return handshakeResponse, nil
}

// setupControlPlane creates and starts the control plane
func (cm *ConnectionManager) setupControlPlane(ctx context.Context, requester ncl.Publisher) error {
	var err error
	cm.controlPlane, err = NewControlPlane(ControlPlaneParams{
		Config:             cm.config,
		Requester:          requester,
		HealthTracker:      cm.healthTracker,
		IncomingSeqTracker: cm.incomingSeqTracker,
		CheckpointName:     cm.incomingCheckpointName,
	})
	if err != nil {
		return fmt.Errorf("failed to create control plane: %w", err)
	}

	if err = cm.controlPlane.Start(ctx); err != nil {
		return fmt.Errorf("failed to start control plane: %w", err)
	}

	return nil
}

// setupDataPlane creates and starts the data plane
func (cm *ConnectionManager) setupDataPlane(ctx context.Context, handshake messages.HandshakeResponse) error {
	var err error
	cm.dataPlane, err = NewDataPlane(DataPlaneParams{
		Config:             cm.config,
		Client:             cm.natsConn,
		LastReceivedSeqNum: handshake.LastComputeSeqNum,
	})
	if err != nil {
		return fmt.Errorf("failed to create data plane: %w", err)
	}

	if err = cm.dataPlane.Start(ctx); err != nil {
		return fmt.Errorf("failed to start data plane: %w", err)
	}

	return nil
}

// maintainConnectionLoop runs a periodic loop that manages the connection lifecycle.
// It handles initial connection, health monitoring, and reconnection with backoff.
func (cm *ConnectionManager) maintainConnectionLoop(ctx context.Context) {
	defer cm.wg.Done()

	// Initial connection attempt
	cm.maintainConnection(ctx)

	// Start periodic connection maintenance
	ticker := cm.config.Clock.Ticker(cm.config.ReconnectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.stopCh:
			return
		case <-ticker.C:
			cm.maintainConnection(ctx)
		}
	}
}

func (cm *ConnectionManager) maintainConnection(ctx context.Context) {
	switch cm.getState() {
	case nclprotocol.Disconnected:
		if err := cm.connect(ctx); err != nil {
			failures := cm.GetHealth().ConsecutiveFailures
			backoffDuration := cm.config.ReconnectBackoff.BackoffDuration(failures)

			log.Error().
				Err(err).
				Int("consecutiveFailures", failures).
				Str("backoffDuration", backoffDuration.String()).
				Msg("Connection attempt failed")

			cm.config.ReconnectBackoff.Backoff(ctx, failures)
		}

	case nclprotocol.Connected:
		cm.checkConnectionHealth()

	default:
		// Ignore other states, such as connecting
	}
}

// checkConnectionHealth verifies the connection is healthy by checking:
// - Recent heartbeat activity
// - NATS connection status
func (cm *ConnectionManager) checkConnectionHealth() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.getState() != nclprotocol.Connected {
		return
	}

	// Consider connection unhealthy if:
	// 1. No heartbeat succeeded within HeartbeatMissFactor intervals
	// 2. NATS connection is closed/draining
	// 3. Health tracker reports a handshake required
	now := cm.config.Clock.Now()
	heartbeatDeadline := now.Add(-time.Duration(cm.config.HeartbeatMissFactor) * cm.config.HeartbeatInterval)

	var reason string
	var unhealthy bool
	health := cm.GetHealth()
	if health.LastSuccessfulHeartbeat.Before(heartbeatDeadline) {
		reason = fmt.Sprintf("no heartbeat for %d intervals", cm.config.HeartbeatMissFactor)
		unhealthy = true
	} else if cm.natsConn.IsClosed() {
		reason = "NATS connection closed"
		unhealthy = true
	} else if cm.healthTracker.IsHandshakeRequired() {
		reason = "handshake required"
		unhealthy = true
	}

	if unhealthy {
		log.Warn().
			Time("lastHeartbeat", health.LastSuccessfulHeartbeat).
			Time("deadline", heartbeatDeadline).
			Int("heartbeatMissFactor", cm.config.HeartbeatMissFactor).
			Str("reason", reason).
			Msg("Connection unhealthy, initiating reconnect")
		cm.transitionState(nclprotocol.Disconnected, fmt.Errorf("connection unhealthy: %s", reason))
	}
}

// transitionState handles state transitions between Connected/Disconnected/Connecting.
// It updates health metrics and notifies registered state change handlers.
func (cm *ConnectionManager) transitionState(newState nclprotocol.ConnectionState, err error) {
	oldState := cm.getState()
	if oldState == newState {
		return
	}

	// Update state tracking
	switch newState {
	case nclprotocol.Connecting:
		cm.healthTracker.MarkConnecting()
	case nclprotocol.Connected:
		cm.healthTracker.MarkConnected()
	case nclprotocol.Disconnected:
		cm.healthTracker.MarkDisconnected(err)
	}

	// Queue state change notification
	select {
	case cm.stateChanges <- stateChange{state: newState, err: err}:
		log.Debug().
			Str("oldState", oldState.String()).
			Str("newState", newState.String()).
			Err(err).
			Msg("Connection state changed")
	default:
		log.Error().Msg("State change notification channel full")
	}
}

func (cm *ConnectionManager) handleStateChanges() {
	defer cm.wg.Done()

	for {
		select {
		case <-cm.stopCh:
			// Process any remaining state changes before exiting
			for {
				select {
				case change, ok := <-cm.stateChanges:
					if !ok {
						return
					}
					cm.processStateChange(change)
				default:
					return
				}
			}
		case change, ok := <-cm.stateChanges:
			if !ok {
				return
			}
			cm.processStateChange(change)
		}
	}
}

// processStateChange handles a single state change notification
func (cm *ConnectionManager) processStateChange(change stateChange) {
	cm.stateHandlersMu.RLock()
	handlers := make([]nclprotocol.ConnectionStateHandler, len(cm.stateHandlers))
	copy(handlers, cm.stateHandlers)
	cm.stateHandlersMu.RUnlock()

	for _, handler := range handlers {
		handler(change.state)
	}
}

// OnStateChange registers a new handler to be called when the connection
// state changes. Handlers are called synchronously when state transitions occur.
func (cm *ConnectionManager) OnStateChange(handler nclprotocol.ConnectionStateHandler) {
	cm.stateHandlersMu.Lock()
	defer cm.stateHandlersMu.Unlock()
	cm.stateHandlers = append(cm.stateHandlers, handler)
}

// GetHealth returns the current health status of the connection including:
// - Timestamps of last successful operations
// - Current connection state
// - Error counts and details
func (cm *ConnectionManager) GetHealth() nclprotocol.ConnectionHealth {
	return cm.healthTracker.GetHealth()
}

// getState returns the current connection state.
func (cm *ConnectionManager) getState() nclprotocol.ConnectionState {
	return cm.GetHealth().CurrentState
}
