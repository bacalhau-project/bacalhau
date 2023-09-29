package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// node discoverer that always returns the same set of nodes
type fixedDiscoverer struct {
	peerIDs []models.NodeInfo
}

func NewFixedDiscoverer(peerIDs ...models.NodeInfo) *fixedDiscoverer {
	return &fixedDiscoverer{
		peerIDs: peerIDs,
	}
}

func (f *fixedDiscoverer) FindNodes(context.Context, models.Job) ([]models.NodeInfo, error) {
	return f.peerIDs, nil
}

func (f *fixedDiscoverer) ListNodes(context.Context) ([]models.NodeInfo, error) {
	return f.peerIDs, nil
}
