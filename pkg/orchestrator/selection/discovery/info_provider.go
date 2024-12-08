package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
)

type discoveredNodesProvider struct {
	discoverer nodes.Lookup
}

func NewDebugInfoProvider(discoverer nodes.Lookup) models.DebugInfoProvider {
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
