package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// node discoverer that always returns the same set of nodes
type fixedDiscoverer struct {
	peerIDs []model.NodeInfo
}

func NewFixedDiscoverer(peerIDs ...model.NodeInfo) *fixedDiscoverer {
	return &fixedDiscoverer{
		peerIDs: peerIDs,
	}
}

func (f *fixedDiscoverer) FindNodes(context.Context, model.Job) ([]model.NodeInfo, error) {
	return f.peerIDs, nil
}

func (f *fixedDiscoverer) ListNodes(context.Context) ([]model.NodeInfo, error) {
	return f.peerIDs, nil
}
