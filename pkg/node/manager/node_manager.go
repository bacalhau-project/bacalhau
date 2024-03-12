package manager

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/concurrency"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	resourceMapLockCount = 32
)

// NodeManager is responsible for managing compute nodes and their
// membership within the cluster through the entire lifecycle. It
// also provides operations for querying and managing compute
// node information.
type NodeManager struct {
	nodeInfo    routing.NodeInfoStore
	resourceMap *concurrency.StripedMap[models.Resources]
}

type NodeManagerParams struct {
	NodeInfo routing.NodeInfoStore
}

// NewNodeManager constructs a new node manager and returns a pointer
// to the structure.
func NewNodeManager(params NodeManagerParams) *NodeManager {
	return &NodeManager{
		resourceMap: concurrency.NewStripedMap[models.Resources](resourceMapLockCount),
		nodeInfo:    params.NodeInfo,
	}
}

//
// ---- Implementation of compute.ManagementEndpoint ----
//

// Register is part of the implementation of the ManagementEndpoint
// interface. It is used to register a compute node with the cluster.
func (n *NodeManager) Register(ctx context.Context, request requests.RegisterRequest) (*requests.RegisterResponse, error) {
	_, err := n.nodeInfo.Get(ctx, request.Info.NodeID)
	if err == nil {
		return &requests.RegisterResponse{
			Accepted: false,
			Reason:   "node already registered",
		}, nil
	}

	// TODO: We will default to PENDING, but once we start filtering on NodeApprovals.APPROVED we will need to
	// make a decision on how this is determined.
	request.Info.Approval = models.NodeApprovals.PENDING

	if err := n.nodeInfo.Add(ctx, request.Info); err != nil {
		return nil, errors.Wrap(err, "failed to save nodeinfo during node registration")
	}

	return &requests.RegisterResponse{
		Accepted: true,
	}, nil
}

// UpdateInfo is part of the implementation of the ManagementEndpoint
// interface. It is used to update the node info for a particular node
func (n *NodeManager) UpdateInfo(ctx context.Context, request requests.UpdateInfoRequest) (*requests.UpdateInfoResponse, error) {
	_, err := n.nodeInfo.Get(ctx, request.Info.NodeID)

	if errors.Is(err, routing.ErrNodeNotFound{}) {
		return &requests.UpdateInfoResponse{
			Accepted: false,
			Reason:   "node not yet registered",
		}, nil
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to get nodeinfo during node registration")
	}

	// TODO(ross): Add a Put endpoint that takes the revision into account
	if err := n.nodeInfo.Add(ctx, request.Info); err != nil {
		return nil, errors.Wrap(err, "failed to save nodeinfo during node registration")
	}

	return &requests.UpdateInfoResponse{
		Accepted: true,
	}, nil
}

// UpdateResources updates the available resources in our in-memory store for each node. This data
// is used to augment information about the available resources for each node.
func (n *NodeManager) UpdateResources(ctx context.Context,
	request requests.UpdateResourcesRequest) (*requests.UpdateResourcesResponse, error) {
	_, err := n.nodeInfo.Get(ctx, request.NodeID)
	if errors.Is(err, routing.ErrNodeNotFound{}) {
		return nil, fmt.Errorf("unable to update resources for missing node: %s", request.NodeID)
	}

	log.Ctx(ctx).Debug().Msg("updating resources availability for node")

	// Update the resources for the node in the stripedmap. This is a thread-safe operation as locking
	// is handled by the stripedmap on a per-bucket basis.
	n.resourceMap.Put(request.NodeID, request.Resources)
	return &requests.UpdateResourcesResponse{}, nil
}

// ---- Implementation of routing.NodeInfoStore ----
func (n *NodeManager) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	return n.nodeInfo.FindPeer(ctx, peerID)
}

func (n *NodeManager) Add(ctx context.Context, nodeInfo models.NodeInfo) error {
	return n.nodeInfo.Add(ctx, nodeInfo)
}

func (n *NodeManager) addResourcesToInfo(ctx context.Context, info *models.NodeInfo) {
	resources, found := n.resourceMap.Get(info.NodeID)
	if found && info.ComputeNodeInfo != nil {
		info.ComputeNodeInfo.AvailableCapacity = resources
	}
}

func (n *NodeManager) Get(ctx context.Context, nodeID string) (models.NodeInfo, error) {
	info, err := n.nodeInfo.Get(ctx, nodeID)
	if err != nil {
		return models.NodeInfo{}, err
	}
	n.addResourcesToInfo(ctx, &info)
	return info, nil
}

func (n *NodeManager) GetByPrefix(ctx context.Context, prefix string) (models.NodeInfo, error) {
	info, err := n.nodeInfo.GetByPrefix(ctx, prefix)
	if err != nil {
		return models.NodeInfo{}, err
	}
	n.addResourcesToInfo(ctx, &info)
	return info, nil
}

func (n *NodeManager) List(ctx context.Context, filters ...routing.NodeInfoFilter) ([]models.NodeInfo, error) {
	items, err := n.nodeInfo.List(ctx, filters...)
	if err != nil {
		return nil, err
	}

	for i := range items {
		n.addResourcesToInfo(ctx, &items[i])
	}

	return items, nil
}

func (n *NodeManager) Delete(ctx context.Context, nodeID string) error {
	return n.nodeInfo.Delete(ctx, nodeID)
}

// ---- Implementation of node actions ----

// Approve is used to approve a node for joining the cluster along with a specific
// reason for the approval (for audit). The return values denote success and any
// failure of the operation as a human readable string.
func (n *NodeManager) Approve(ctx context.Context, nodeID string, reason string) (bool, string) {
	info, err := n.nodeInfo.Get(ctx, nodeID)
	if err != nil {
		return false, "node not found"
	}

	if info.Approval == models.NodeApprovals.APPROVED {
		return false, "node already approved"
	}

	info.Approval = models.NodeApprovals.APPROVED
	log.Ctx(ctx).Info().Str("reason", reason).Msgf("node %s approved", nodeID)

	if err := n.nodeInfo.Add(ctx, info); err != nil {
		return false, "failed to save nodeinfo during node approval"
	}

	return true, ""
}

// Reject is used to reject a node from joining the cluster along with a specific
// reason for the rejection (for audit). The return values denote success and any
// failure of the operation as a human readable string.
func (n *NodeManager) Reject(ctx context.Context, nodeID string, reason string) (bool, string) {
	info, err := n.nodeInfo.Get(ctx, nodeID)
	if err != nil {
		return false, "node not found"
	}

	if info.Approval == models.NodeApprovals.REJECTED {
		return false, "node already rejected"
	}

	info.Approval = models.NodeApprovals.REJECTED
	log.Ctx(ctx).Info().Str("reason", reason).Msgf("node %s rejected", nodeID)

	if err := n.nodeInfo.Add(ctx, info); err != nil {
		return false, "failed to save nodeinfo during node rejection"
	}

	return true, ""
}

var _ compute.ManagementEndpoint = (*NodeManager)(nil)
var _ routing.NodeInfoStore = (*NodeManager)(nil)
