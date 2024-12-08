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
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
)

// ComputeManager handles the lifecycle and state management of all compute nodes
// connected to this orchestrator. It tracks node health, handles control messages,
// and coordinates data plane communication.
type ComputeManager struct {
	config   Config
	natsConn *nats.Conn

	// Control plane components
	responder ncl.Responder

	// Node management
	nodeManager nodes.Manager
	dataPlanes  sync.Map // map[string]*DataPlane

	// Control channels
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewComputeManager creates a new compute manager with the given configuration.
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

// Start initializes the manager and begins processing compute node connections
func (cm *ComputeManager) Start(ctx context.Context) error {
	var err error

	// Create NATS connection
	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create control plane responder
	cm.responder, err = ncl.NewResponder(cm.natsConn, ncl.ResponderConfig{
		Name:              "orchestrator-control",
		MessageRegistry:   cm.config.MessageRegistry,
		MessageSerializer: cm.config.MessageSerializer,
		Subject:           core.NatsSubjectComputeOutCtrl("*"), // Subscribe to all compute nodes
	})
	if err != nil {
		return fmt.Errorf("failed to create control responder: %w", err)
	}

	// Register control message handlers
	err = errors.Join(
		cm.responder.Listen(ctx, messages.HandshakeRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleHandshakeRequest)),
		cm.responder.Listen(ctx, messages.HeartbeatRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleHeartbeatRequest)),
		cm.responder.Listen(ctx, messages.NodeInfoUpdateRequestMessageType,
			ncl.RequestHandlerFunc(cm.handleNodeInfoUpdateRequest)),
	)
	if err != nil {
		return fmt.Errorf("failed to register message handlers: %w", err)
	}

	// Register for connection state changes
	cm.nodeManager.OnConnectionStateChange(cm.handleConnectionStateChange)

	return nil
}

// Stop gracefully shuts down the manager and all compute nodes
func (cm *ComputeManager) Stop(ctx context.Context) error {
	close(cm.stopCh)

	// Stop responder first to prevent new connections
	if cm.responder != nil {
		if err := cm.responder.Close(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to close responder")
		}
	}

	// Stop all data planes
	cm.dataPlanes.Range(func(key, value interface{}) bool {
		if dataPlane, ok := value.(*DataPlane); ok {
			if err := dataPlane.Stop(ctx); err != nil {
				log.Error().Err(err).Str("node", key.(string)).
					Msg("Failed to stop data plane")
			}
		}
		return true
	})

	// Wait for all goroutines to finish
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

// handleHandshakeRequest processes compute node handshake requests
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

	// Create data plane if node is accepted
	if err = cm.setupDataPlane(ctx, request.NodeInfo, request.LastOrchestratorSeqNum); err != nil {
		return nil, fmt.Errorf("failed to setup data plane: %w", err)
	}

	return envelope.NewMessage(response), nil
}

func (cm *ComputeManager) setupDataPlane(ctx context.Context, nodeInfo models.NodeInfo, lastReceivedSeqNum uint64) error {
	// Create data plane starting from lastSeqNum
	dataPlane, err := NewDataPlane(DataPlaneConfig{
		NodeID:                nodeInfo.ID(),
		Client:                cm.natsConn,
		MessageRegistry:       cm.config.MessageRegistry,
		MessageSerializer:     cm.config.MessageSerializer,
		MessageHandler:        cm.config.DataPlaneMessageHandler,
		MessageCreatorFactory: cm.config.DataPlaneMessageCreatorFactory,
		EventStore:            cm.config.EventStore,
		StartSeqNum:           lastReceivedSeqNum,
	})
	if err != nil {
		return err
	}

	// Atomically replace old with new, stopping old if it exists
	if existing, loaded := cm.dataPlanes.Swap(nodeInfo.ID(), dataPlane); loaded {
		if dp, ok := existing.(*DataPlane); ok {
			if err = dp.Stop(context.TODO()); err != nil {
				log.Error().Err(err).Str("node", nodeInfo.ID()).
					Msg("Failed to stop existing data plane")
			}
		}
	}

	// Start data plane
	return dataPlane.Start(context.TODO())
}

// handleHeartbeatRequest processes compute node heartbeat requests
func (cm *ComputeManager) handleHeartbeatRequest(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
	request := msg.Payload.(*messages.HeartbeatRequest)

	// Verify data plane exists
	dataPlane, ok := cm.dataPlanes.Load(request.NodeID)
	if !ok {
		// Return error asking node to reconnect since it has no active data plane
		return nil, fmt.Errorf("no active data plane - handshake required")
	}

	// Process through node manager
	response, err := cm.nodeManager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest:  *request,
		LastComputeSeqNum: dataPlane.(*DataPlane).incomingSequenceTracker.GetLastSeqNum(),
	})
	if err != nil {
		return nil, err
	}

	return envelope.NewMessage(response), nil
}

// handleNodeInfoUpdateRequest processes node info update requests
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
					log.Error().Err(err).Str("node", event.NodeID).
						Msg("Failed to stop data plane for disconnected node")
				}
			}
		}
	}
}
