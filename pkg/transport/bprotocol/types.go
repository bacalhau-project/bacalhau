package bprotocol

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

// ManagementEndpoint is the transport-based interface for compute nodes to
// register with the requester node, update information and perform heartbeats.
type ManagementEndpoint interface {
	// Register registers a compute node with the requester node.
	Register(context.Context, legacy.RegisterRequest) (*legacy.RegisterResponse, error)
	// UpdateInfo sends an update of node info to the requester node
	UpdateInfo(context.Context, legacy.UpdateInfoRequest) (*legacy.UpdateInfoResponse, error)
	// UpdateResources updates the resources currently in use by a specific node
	UpdateResources(context.Context, legacy.UpdateResourcesRequest) (*legacy.UpdateResourcesResponse, error)
}
