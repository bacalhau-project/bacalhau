package manager

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

const (
	resourceMapLockCount = 32
)

// NodeManager is responsible for managing compute nodes and their
// membership within the cluster through the entire lifecycle. It
// also provides operations for querying and managing compute
// node information.
type NodeManager struct {
	store                routing.NodeInfoStore
	resourceMap          *concurrency.StripedMap[models.Resources]
	heartbeats           *heartbeat.HeartbeatServer
	defaultApprovalState models.NodeMembershipState
}

type NodeManagerParams struct {
	NodeInfo             routing.NodeInfoStore
	Heartbeats           *heartbeat.HeartbeatServer
	DefaultApprovalState models.NodeMembershipState
}

// NewNodeManager constructs a new node manager and returns a pointer
// to the structure.
func NewNodeManager(params NodeManagerParams) *NodeManager {
	return &NodeManager{
		resourceMap:          concurrency.NewStripedMap[models.Resources](resourceMapLockCount),
		store:                params.NodeInfo,
		heartbeats:           params.Heartbeats,
		defaultApprovalState: params.DefaultApprovalState,
	}
}

func (n *NodeManager) Start(ctx context.Context) error {
	if n.heartbeats != nil {
		err := n.heartbeats.Start(ctx)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to start heartbeat server")
			return err
		}
	}

	log.Ctx(ctx).Info().Msg("Node manager started")

	return nil
}

//
// ---- Implementation of compute.ManagementEndpoint ----
//

// Register is part of the implementation of the ManagementEndpoint
// interface. It is used to register a compute node with the cluster.
func (n *NodeManager) Register(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	existing, err := n.store.Get(ctx, request.Info.NodeID)
	if err == nil {
		// If we have already seen this node and rejected it, then let the node know
		if existing.Membership == models.NodeMembership.REJECTED {
			return &requests.RegisterResponse{
				Accepted: false,
				Reason:   "node has been rejected",
			}, nil
		}

		// Otherwise we'll allow the registration, but let the compute node
		// that it has already been registered on a previous occasion.
		return &requests.RegisterResponse{
			Accepted: true,
			Reason:   "node already registered",
		}, nil
	}

	if err := n.store.Add(ctx, models.NodeState{
		Info:       request.Info,
		Membership: n.defaultApprovalState,
		// NB(forrest): by virtue of a compute node calling this endpoint we can consider it connected
		Connection: models.NodeStates.CONNECTED,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to save nodestate during node registration")
	}

	return &requests.RegisterResponse{
		Accepted: true,
	}, nil
}

// UpdateInfo is part of the implementation of the ManagementEndpoint
// interface. It is used to update the node state for a particular node
func (n *NodeManager) UpdateInfo(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	existing, err := n.store.Get(ctx, request.Info.NodeID)

	if errors.Is(err, routing.ErrNodeNotFound{}) {
		return &requests.UpdateInfoResponse{
			Accepted: false,
			Reason:   "node not yet registered",
		}, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get nodestate during node registration")
	}

	if existing.Membership == models.NodeMembership.REJECTED {
		return &requests.UpdateInfoResponse{
			Accepted: false,
			Reason:   "node registration rejected",
		}, nil
	}

	// TODO: Add a Put endpoint that takes the revision into account?
	if err := n.store.Add(ctx, models.NodeState{
		Info: request.Info,
		// the nodes approval state is assumed to be approved here, but re-use existing state
		Membership: existing.Membership,
		// TODO can we assume the node is connected here?
		Connection: models.NodeStates.CONNECTED,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to save nodestate during node registration")
	}

	return &requests.UpdateInfoResponse{
		Accepted: true,
	}, nil
}

// UpdateResources updates the available resources in our in-memory store for each node. This data
// is used to augment information about the available resources for each node.
func (n *NodeManager) UpdateResources(ctx context.Context,
	request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error) {
	existing, err := n.store.Get(ctx, request.NodeID)
	if errors.Is(err, routing.ErrNodeNotFound{}) {
		return nil, fmt.Errorf("unable to update resources for missing node: %s", request.NodeID)
	}

	if existing.Membership == models.NodeMembership.REJECTED {
		log.Ctx(ctx).Debug().Msg("not updating resources for rejected node ")
		return &requests.UpdateResourcesResponse{}, nil
	}

	log.Ctx(ctx).Debug().Msg("updating resources availability for node")

	// Update the resources for the node in the stripedmap. This is a thread-safe operation as locking
	// is handled by the stripedmap on a per-bucket basis.
	n.resourceMap.Put(request.NodeID, request.Resources)
	return &requests.UpdateResourcesResponse{}, nil
}

// ---- Implementation of routing.NodeInfoStore ----
func (n *NodeManager) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	return n.store.FindPeer(ctx, peerID)
}

func (n *NodeManager) Add(ctx context.Context, nodeInfo models.NodeState) error {
	return n.store.Add(ctx, nodeInfo)
}

func (n *NodeManager) addToInfo(ctx context.Context, state *models.NodeState) {
	resources, found := n.resourceMap.Get(state.Info.NodeID)
	if found && state.Info.ComputeNodeInfo != nil {
		state.Info.ComputeNodeInfo.AvailableCapacity = resources
	}

	if n.heartbeats != nil {
		n.heartbeats.UpdateNodeInfo(state)
	}
}

func (n *NodeManager) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	nodeState, err := n.store.Get(ctx, nodeID)
	if err != nil {
		return models.NodeState{}, err
	}
	n.addToInfo(ctx, &nodeState)
	return nodeState, nil
}

func (n *NodeManager) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	state, err := n.store.GetByPrefix(ctx, prefix)
	if err != nil {
		return models.NodeState{}, err
	}
	n.addToInfo(ctx, &state)
	return state, nil
}

func (n *NodeManager) List(ctx context.Context, filters ...routing.NodeStateFilter) ([]models.NodeState, error) {
	items, err := n.store.List(ctx, filters...)
	if err != nil {
		return nil, err
	}

	for i := range items {
		n.addToInfo(ctx, &items[i])
	}

	return items, nil
}

func (n *NodeManager) Delete(ctx context.Context, nodeID string) error {
	return n.store.Delete(ctx, nodeID)
}

// ---- Implementation of node actions ----

// Approve is used to approve a node for joining the cluster along with a specific
// reason for the approval (for audit). The return values denote success and any
// failure of the operation as a human readable string.
func (n *NodeManager) ApproveAction(ctx context.Context, nodeID string, reason string) (bool, string) {
	state, err := n.store.GetByPrefix(ctx, nodeID)
	if err != nil {
		return false, err.Error()
	}

	if state.Membership == models.NodeMembership.APPROVED {
		return false, "node already approved"
	}

	state.Membership = models.NodeMembership.APPROVED
	log.Ctx(ctx).Info().Str("reason", reason).Msgf("node %s approved", nodeID)

	if err := n.store.Add(ctx, state); err != nil {
		return false, "failed to save nodestate during node approval"
	}

	return true, ""
}

// Reject is used to reject a node from joining the cluster along with a specific
// reason for the rejection (for audit). The return values denote success and any
// failure of the operation as a human readable string.
func (n *NodeManager) RejectAction(ctx context.Context, nodeID string, reason string) (bool, string) {
	state, err := n.store.GetByPrefix(ctx, nodeID)
	if err != nil {
		return false, err.Error()
	}

	if state.Membership == models.NodeMembership.REJECTED {
		return false, "node already rejected"
	}

	state.Membership = models.NodeMembership.REJECTED
	log.Ctx(ctx).Info().Str("reason", reason).Msgf("node %s rejected", nodeID)

	if err := n.store.Add(ctx, state); err != nil {
		return false, "failed to save nodestate during node rejection"
	}

	return true, ""
}

// Reject is used to reject a node from joining the cluster along with a specific
// reason for the rejection (for audit). The return values denote success and any
// failure of the operation as a human readable string.
func (n *NodeManager) DeleteAction(ctx context.Context, nodeID string, reason string) (bool, string) {
	state, err := n.store.GetByPrefix(ctx, nodeID)
	if err != nil {
		return false, err.Error()
	}

	if err := n.store.Delete(ctx, state.Info.NodeID); err != nil {
		return false, fmt.Sprintf("failed to delete nodestate: %s", err)
	}

	return true, ""
}

var _ compute.ManagementEndpoint = (*NodeManager)(nil)
var _ routing.NodeInfoStore = (*NodeManager)(nil)
