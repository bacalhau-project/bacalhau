package node

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type Requester struct {
	// Visible for testing
	Endpoint   requester.Endpoint
	EndpointV2 *orchestrator.BaseEndpoint
	JobStore   jobstore.Store
	// We need a reference to the node info store until libp2p is removed
	NodeInfoStore      routing.NodeInfoStore
	NodeDiscoverer     orchestrator.NodeDiscoverer
	nodeManager        *manager.NodeManager
	localCallback      compute.Callback
	cleanupFunc        func(ctx context.Context)
	debugInfoProviders []model.DebugInfoProvider
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
