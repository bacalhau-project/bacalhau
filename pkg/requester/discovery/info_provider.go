package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
)

type discoveredNodesProvider struct {
	discoverer requester.NodeDiscoverer
}

func NewDebugInfoProvider(discoverer requester.NodeDiscoverer) model.DebugInfoProvider {
	return &discoveredNodesProvider{discoverer: discoverer}
}

// GetDebugInfo implements model.DebugInfoProvider
func (p *discoveredNodesProvider) GetDebugInfo(ctx context.Context) (info model.DebugInfo, err error) {
	nodes, err := p.discoverer.ListNodes(ctx)
	info.Component = "DiscoveredNodes"
	info.Info = nodes
	return info, err
}

var _ model.DebugInfoProvider = (*discoveredNodesProvider)(nil)
