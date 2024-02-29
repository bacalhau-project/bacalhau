package manager

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/requests"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
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

// Register is part of the implementation of the RegistrationEndpoint
// interface. It is used to register a compute node with the cluster.
func (n *NodeManager) Register(ctx context.Context, request requests.RegisterRequest) error {
	if err := n.nodeInfo.Add(ctx, request.Info); err != nil {
		return errors.Wrap(err, "failed to save nodeinfo during node registration")
	}

	return nil
}

var _ requester.RegistrationEndpoint = (*NodeManager)(nil)
