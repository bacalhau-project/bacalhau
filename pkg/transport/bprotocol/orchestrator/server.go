package orchestrator

import (
	"context"
	"reflect"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

type Server struct {
	nodeManager nodes.Manager
	resources   sync.Map // map[string]nodeResources
	// Track nodes that failed registration to avoid infinite retries
	failedRegistrations sync.Map // map[string]struct{}
	reregistrationMu    sync.Mutex
}

type nodeResources struct {
	availableCapacity models.Resources
	queueUsedCapacity models.Resources
	lastUpdate        time.Time
}

func NewServer(manager nodes.Manager) *Server {
	return &Server{
		nodeManager:         manager,
		failedRegistrations: sync.Map{},
		reregistrationMu:    sync.Mutex{},
	}
}

func (h *Server) ShouldProcess(ctx context.Context, message *envelope.Message) bool {
	return message.Metadata.Get(envelope.KeyMessageType) == legacy.HeartbeatMessageType
}

// HandleMessage processes NCL messages and routes them to the appropriate handler
func (h *Server) HandleMessage(ctx context.Context, message *envelope.Message) error {
	payload, ok := message.GetPayload(legacy.Heartbeat{})
	if !ok {
		return envelope.NewErrUnexpectedPayloadType(
			reflect.TypeOf(legacy.Heartbeat{}).String(),
			reflect.TypeOf(message.Payload).String())
	}

	return h.Heartbeat(ctx, payload.(legacy.Heartbeat))
}

// Register handles compute node registration requests
func (h *Server) Register(ctx context.Context, request legacy.RegisterRequest) (*legacy.RegisterResponse, error) {
	// Check if the node supports NCLv1 protocol
	for _, protocol := range request.Info.SupportedProtocols {
		if protocol == models.ProtocolNCLV1 {
			return &legacy.RegisterResponse{
				Accepted: false,
				Reason:   bprotocol.ErrUpgradeAvailable.Error(),
			}, nil
		}
	}

	resp, err := h.nodeManager.Handshake(ctx, messages.HandshakeRequest{
		NodeInfo:  request.Info,
		StartTime: time.Now(),
	})

	if err != nil {
		return nil, err
	}

	// Store the latest resource update
	h.resources.Store(request.Info.ID(), nodeResources{
		availableCapacity: request.Info.ComputeNodeInfo.AvailableCapacity,
		queueUsedCapacity: request.Info.ComputeNodeInfo.QueueUsedCapacity,
		lastUpdate:        time.Now(),
	})

	return &legacy.RegisterResponse{
		Accepted: resp.Accepted,
		Reason:   resp.Reason,
	}, nil
}

// UpdateInfo handles compute node info update requests
func (h *Server) UpdateInfo(ctx context.Context, request legacy.UpdateInfoRequest) (*legacy.UpdateInfoResponse, error) {
	_, err := h.nodeManager.UpdateNodeInfo(ctx, messages.UpdateNodeInfoRequest{
		NodeInfo: request.Info,
	})

	if err != nil {
		return nil, err
	}

	return &legacy.UpdateInfoResponse{
		Accepted: true,
	}, nil
}

// UpdateResources stores the latest resource information for a node and forwards it to the manager
func (h *Server) UpdateResources(ctx context.Context, request legacy.UpdateResourcesRequest) (*legacy.UpdateResourcesResponse, error) {
	// Store the latest resource update
	h.resources.Store(request.NodeID, nodeResources{
		availableCapacity: request.AvailableCapacity,
		queueUsedCapacity: request.QueueUsedCapacity,
		lastUpdate:        time.Now(),
	})

	// Forward to node manager
	_, err := h.nodeManager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: messages.HeartbeatRequest{
			NodeID:            request.NodeID,
			AvailableCapacity: request.AvailableCapacity,
			QueueUsedCapacity: request.QueueUsedCapacity,
		},
	})

	if err != nil {
		return nil, err
	}

	return &legacy.UpdateResourcesResponse{}, nil
}

func (h *Server) Heartbeat(ctx context.Context, request legacy.Heartbeat) error {
	// Create base heartbeat request
	heartbeat := messages.HeartbeatRequest{
		NodeID: request.NodeID,
	}

	// Enrich with latest resource information if available
	if resources, ok := h.resources.Load(request.NodeID); ok {
		res := resources.(nodeResources)
		heartbeat.AvailableCapacity = res.availableCapacity
		heartbeat.QueueUsedCapacity = res.queueUsedCapacity
	}

	_, err := h.nodeManager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
		HeartbeatRequest: heartbeat,
	})

	// If handshake required error, attempt re-registration
	if err != nil && bacerrors.IsErrorWithCode(err, nodes.HandshakeRequired) {
		if err := h.tryReregisterNode(ctx, request.NodeID); err != nil {
			return err
		}

		// Retry heartbeat after successful re-registration
		_, err = h.nodeManager.Heartbeat(ctx, nodes.ExtendedHeartbeatRequest{
			HeartbeatRequest: heartbeat,
		})
	}

	return err
}

// tryReregisterNode attempts to re-register a node that failed a heartbeat
// due to missing handshake. Returns error if registration fails.
func (h *Server) tryReregisterNode(ctx context.Context, nodeID string) error {
	// Check if we've already tried and failed to register this node
	if _, failed := h.failedRegistrations.Load(nodeID); failed {
		return nodes.NewErrHandshakeRequired(nodeID)
	}

	// Prevent concurrent re-registration attempts
	h.reregistrationMu.Lock()
	defer h.reregistrationMu.Unlock()

	// check again if we've already tried and failed to register this node
	if _, failed := h.failedRegistrations.Load(nodeID); failed {
		return nodes.NewErrHandshakeRequired(nodeID)
	}

	// Get the node info from the manager
	state, err := h.nodeManager.Get(ctx, nodeID)
	if err != nil {
		// If node not found, mark as failed registration and don't retry
		h.failedRegistrations.Store(nodeID, struct{}{})
		log.Error().Err(err).Str("node", nodeID).Msg("Failed to get node info for re-registration")
		return err
	}

	// Attempt to re-register the node
	resp, err := h.Register(ctx, legacy.RegisterRequest{
		Info: state.Info,
	})
	if err != nil {
		h.failedRegistrations.Store(nodeID, struct{}{})
		log.Error().Err(err).Str("node", nodeID).Msg("Failed to re-register node")
		return err
	}

	if !resp.Accepted {
		h.failedRegistrations.Store(nodeID, struct{}{})
		log.Error().Str("node", nodeID).Str("reason", resp.Reason).Msg("Node re-registration rejected")
		return nodes.NewErrHandshakeRequired(nodeID)
	}

	log.Info().Str("node", nodeID).Msg("Successfully re-registered legacy node")
	return nil
}

var _ ncl.MessageHandler = (*Server)(nil)
var _ bprotocol.ManagementEndpoint = (*Server)(nil)
