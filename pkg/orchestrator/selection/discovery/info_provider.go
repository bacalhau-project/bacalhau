package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type discoveredNodesProvider struct {
	discoverer orchestrator.NodeDiscoverer
}

func NewDebugInfoProvider(discoverer orchestrator.NodeDiscoverer) models.DebugInfoProvider {
	return &discoveredNodesProvider{discoverer: discoverer}
}

// GetDebugInfo implements models.DebugInfoProvider
func (p *discoveredNodesProvider) GetDebugInfo(ctx context.Context) (info models.DebugInfo, err error) {
	nodes, err := p.discoverer.List(ctx)
	info.Component = "DiscoveredNodes"
	info.Info = nodes
	return info, err
}

var _ models.DebugInfoProvider = (*discoveredNodesProvider)(nil)
