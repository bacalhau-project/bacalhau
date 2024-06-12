package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TODO rename this interface to NodeStore, it tracks more than their info
type NodeInfoStore interface {
	// Add adds a node info to the repo.
	Add(ctx context.Context, nodeInfo models.NodeState) error

	// Get returns the node info for the given node ID.
	Get(ctx context.Context, nodeID string) (models.NodeState, error)

	// GetByPrefix returns the node info for the given node ID.
	// Supports both full and short node IDs.
	GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error)

	// List returns a list of nodes
	List(ctx context.Context, filters ...NodeStateFilter) ([]models.NodeState, error)

	// Delete deletes a node info from the repo.
	Delete(ctx context.Context, nodeID string) error
}

// NodeStateFilter is a function that filters node state
// when listing nodes. It returns true if the node state
// should be returned, and false if the node state should
// be ignored.
type NodeStateFilter func(models.NodeState) bool
