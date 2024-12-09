package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
)

// ComputeManager handles the lifecycle and state management of all compute nodes
// connected to this orchestrator. It is responsible for:
// - Processing compute node handshakes and connections
// - Managing individual node data planes
// - Coordinating message flow between orchestrator and compute nodes
// - Tracking node health and connection state
type ComputeManager struct {
	config Config

	// Core components
	natsConn   *nats.Conn    // NATS connection
	responder  ncl.Responder // Handles control plane requests
	dataPlanes sync.Map      // map[string]*DataPlane

	// Node management
	nodeManager nodes.Manager // Tracks node state and health

	// Lifecycle management
	stopCh chan struct{}  // Signals background goroutines to stop
	wg     sync.WaitGroup // Tracks active background goroutines
}

// NewComputeManager creates a new compute manager with the given configuration.
// The manager must be started with Start() before it begins processing connections.
func NewComputeManager(cfg Config) (*ComputeManager, error) {
	cfg.setDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &ComputeManager{
		config:      cfg,
		nodeManager: cfg.NodeManager,
		stopCh:      make(chan struct{}),
	}, nil
}

// Start initializes the manager and begins processing compute node connections.
// This includes:
// 1. Creating NATS connection
// 2. Setting up control plane responder
// 3. Registering message handlers
// 4. Setting up node state change handling
func (cm *ComputeManager) Start(ctx context.Context) error {
	var err error

	// Create NATS connection
	if err = cm.setupTransport(ctx); err != nil {
		return err
	}

	// Set up control plane responder
	if err = cm.setupControlPlane(ctx); err != nil {
		return err
	}

	// Register for node state changes
	cm.nodeManager.OnConnectionStateChange(cm.handleConnectionStateChange)

	return nil
}

func (cm *ComputeManager) setupTransport(ctx context.Context) error {
	var err error
	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}
	return nil
}

func (cm *ComputeManager) setupControlPlane(ctx context.Context) error {
	var err error

	// Create responder for control messages
	cm.responder, err = ncl.NewResponder(cm.natsConn, ncl.ResponderConfig{
		Name:              "orchestrator-control",
		MessageRegistry:   cm.config.MessageRegistry,
		MessageSerializer: cm.config.MessageSerializer,
		Subject:           nclprotocol.NatsSubjectOrchestratorInCtrl(),
	})
	if err != nil {
		return fmt.Errorf("create control responder: %w", err)
	}

	// Register control message handlers
	return errors.Join(
		cm.responder.Listen(ctx, messages.HandshakeRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleHandshakeRequest)),
		cm.responder.Listen(ctx, messages.HeartbeatRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleHeartbeatRequest)),
		cm.responder.Listen(ctx, messages.NodeInfoUpdateRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleNodeInfoUpdateRequest)),
	)
}

// Stop gracefully shuts down the manager and all compute node connections.
// It ensures proper cleanup by:
// 1. Stopping the control plane responder
// 2. Stopping all data planes
// 3. Waiting for background goroutines to complete
func (cm *ComputeManager) Stop(ctx context.Context) error {
	close(cm.stopCh)

	var errs error

	// Stop responder first to prevent new connections
	if cm.responder != nil {
		if err := cm.responder.Close(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("close responder: %w", err))
		}
		cm.responder = nil
	}

	// Stop all data planes
	cm.dataPlanes.Range(func(key, value interface{}) bool {
		nodeID := key.(string)
		if dataPlane, ok := value.(*DataPlane); ok {
			if err := dataPlane.Stop(ctx); err != nil {
				errs = errors.Join(errs,
					fmt.Errorf("stop data plane for node %s: %w", nodeID, err))
			}
		}
		return true
	})

	// Clean up NATS connection
	if cm.natsConn != nil {
		cm.natsConn.Close()
		cm.natsConn = nil
	}

	// Wait for goroutines with timeout
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if errs != nil {
			return fmt.Errorf("shutdown errors: %w", errs)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timeout: %w", ctx.Err())
	}
}

// handleHandshakeRequest processes incoming handshake requests from compute nodes.
// For each new node, it:
// 1. Validates the request through node manager
// 2. Creates a new data plane if accepted
// 3. Returns handshake response with connection details
func (cm *ComputeManager) handleHandshakeRequest(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
	request := msg.Payload.(*messages.HandshakeRequest)

	// Process handshake through node manager
	response, err := cm.nodeManager.Handshake(ctx, *request)
	if err != nil {
		return nil, err
	}

	if !response.Accepted {
		return envelope.NewMessage(response), nil
	}

	// Create data plane for accepted node
	if err = cm.setupDataPlane(ctx, request.NodeInfo, request.LastOrchestratorSeqNum); err != nil {
		return nil, fmt.Errorf("setup data plane failed: %w", err)
	}

	return envelope.NewMessage(response), nil
}

// setupDataPlane creates and starts a new data plane for a compute node.
// If a data plane already exists for the node, it is gracefully stopped
// and replaced with the new one.
func (cm *ComputeManager) setupDataPlane(
	ctx context.Context,
	nodeInfo models.NodeInfo,
	lastReceivedSeqNum uint64,
) error {
	// Create new data plane configuration
	dataPlane, err := NewDataPlane(DataPlaneConfig{
		NodeID:                nodeInfo.ID(),
		Client:                cm.natsConn,
		MessageRegistry:       cm.config.MessageRegistry,
		MessageSerializer:     cm.config.MessageSerializer,
		MessageHandler:        cm.config.DataPlaneMessageHandler,
		MessageCreatorFactory: cm.config.DataPlaneMessageCreatorFactory,
		EventStore:            cm.config.EventStore,
		StartSeqNum:           lastReceivedSeqNum,
		DispatcherConfig:      cm.config.DispatcherConfig,
	})
	if err != nil {
		return err
	}

	// Atomically replace old with new, stopping old if it exists
	if existing, loaded := cm.dataPlanes.Swap(nodeInfo.ID(), dataPlane); loaded {
		if dp, ok := existing.(*DataPlane); ok {
			if err = dp.Stop(context.TODO()); err != nil {
				log.Error().
					Err(err).
					Str("nodeID", nodeInfo.ID()).
					Msg("Failed to stop existing data plane")
			}
		}
	}

	// Start new data plane
	if err = dataPlane.Start(ctx); err != nil {
		cm.dataPlanes.Delete(nodeInfo.ID())
		return fmt.Errorf("start data plane: %w", err)
	}

	return nil
}

// handleHeartbeatRequest processes heartbeat messages from compute nodes.
// It verifies the node has an active data plane and updates health tracking.
func (cm *ComputeManager) handleHeartbeatRequest(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
	request := msg.Payload.(*messages.HeartbeatRequest)

	// Verify data plane exists
	dataPlane, exists := cm.getDataPlane(request.NodeID)
	if !exists {
		return nil, fmt.Errorf("no active data plane for node %s - handshake required", request.NodeID)
	}

	// Process through node manager with sequence info
	response, err := cm.nodeManager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest:  *request,
		LastComputeSeqNum: dataPlane.GetLastProcessedSequence(),
	})
	if err != nil {
		return nil, err
	}

	return envelope.NewMessage(response), nil
}

// handleNodeInfoUpdateRequest processes node info updates from compute nodes.
// It verifies the node has an active data plane before accepting updates.
func (cm *ComputeManager) handleNodeInfoUpdateRequest(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
	request := msg.Payload.(*messages.UpdateNodeInfoRequest)

	// Verify data plane exists
	if _, ok := cm.dataPlanes.Load(request.NodeInfo.ID()); !ok {
		// Return error asking node to reconnect since it has no active data plane
		return nil, fmt.Errorf("no active data plane - handshake required")
	}

	// Process through node manager
	response, err := cm.nodeManager.UpdateNodeInfo(ctx, *request)
	if err != nil {
		return nil, err
	}

	return envelope.NewMessage(response), nil
}

// handleConnectionStateChange responds to node connection state changes
func (cm *ComputeManager) handleConnectionStateChange(event nodes.NodeConnectionEvent) {
	// If node disconnected, stop and remove data plane
	if event.Current == models.NodeStates.DISCONNECTED {
		if dataPlane, ok := cm.dataPlanes.LoadAndDelete(event.NodeID); ok {
			if dp, ok := dataPlane.(*DataPlane); ok {
				if err := dp.Stop(context.Background()); err != nil {
					log.Error().Err(err).
						Str("nodeID", event.NodeID).
						Msg("Failed to stop data plane for disconnected node")
				}
			}
		}
	}
}

// getDataPlane safely retrieves the data plane for a node if it exists
func (cm *ComputeManager) getDataPlane(nodeID string) (*DataPlane, bool) {
	if value, ok := cm.dataPlanes.Load(nodeID); ok {
		if dataPlane, ok := value.(*DataPlane); ok {
			return dataPlane, true
		}
	}
	return nil, false
}
