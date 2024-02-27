package manager

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
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

// Register is part of the implmentation of the RegistrationEndpoint
// interface. It is used to register a compute node with the cluster.
func (n *NodeManager) Register(ctx context.Context) error {
	// TODO: Construct a node info object and register it with the
	// node info store.

	// TODO: Set the node approval state if known
	return nil
}

var _ requester.RegistrationEndpoint = (*NodeManager)(nil)
