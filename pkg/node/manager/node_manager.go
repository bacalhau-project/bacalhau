package manager

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/pkg/errors"
)

// NodeManager is responsible for managing compute nodes and their
// membership within the cluster through the entire lifecycle. It
// also provides operations for querying and managing compute
// node information.
type NodeManager struct {
	nodeInfo routing.NodeInfoStore
}

type NodeManagerParams struct {
	NodeInfo routing.NodeInfoStore
}

// NewNodeManager constructs a new node manager and returns a pointer
// to the structure.
func NewNodeManager(params NodeManagerParams) *NodeManager {
	return &NodeManager{
		nodeInfo: params.NodeInfo,
	}
}

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

var _ compute.ManagementEndpoint = (*NodeManager)(nil)
