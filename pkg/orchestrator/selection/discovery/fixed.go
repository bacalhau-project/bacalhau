package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// node discoverer that always returns the same set of nodes
type fixedDiscoverer struct {
	peerIDs []models.NodeState
}

func NewFixedDiscoverer(peerIDs ...models.NodeState) *fixedDiscoverer {
	return &fixedDiscoverer{
		peerIDs: peerIDs,
	}
}

func (f *fixedDiscoverer) FindNodes(context.Context, models.Job) ([]models.NodeState, error) {
	return f.peerIDs, nil
}

func (f *fixedDiscoverer) ListNodes(context.Context) ([]models.NodeState, error) {
	return f.peerIDs, nil
}
