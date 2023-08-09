package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type StoreNodeDiscovererParams struct {
	Store routing.NodeInfoStore
}

type StoreNodeDiscoverer struct {
	store routing.NodeInfoStore
}

func NewStoreNodeDiscoverer(params StoreNodeDiscovererParams) *StoreNodeDiscoverer {
	return &StoreNodeDiscoverer{
		store: params.Store,
	}
}

// FindNodes returns the nodes that support the job's execution engine, and have enough TOTAL capacity to run the job.
func (d *StoreNodeDiscoverer) FindNodes(ctx context.Context, job model.Job) ([]model.NodeInfo, error) {
	// filter nodes that support the job's engine
	return d.store.ListForEngine(ctx, job.Spec.EngineSpec.Engine())
}

// ListNodes implements orchestrator.NodeDiscoverer
func (d *StoreNodeDiscoverer) ListNodes(ctx context.Context) ([]model.NodeInfo, error) {
	return d.store.List(ctx)
}

// compile time check that StoreNodeDiscoverer implements NodeDiscoverer
var _ orchestrator.NodeDiscoverer = (*StoreNodeDiscoverer)(nil)
