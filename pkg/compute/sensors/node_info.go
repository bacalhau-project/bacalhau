package sensors

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type NodeDebugInfoProviderParams struct {
	Name             string
	NodeInfoProvider model.NodeInfoProvider
}

// NodeDebugInfoProvider is a debug info provider that returns the available Node of the node.
type NodeDebugInfoProvider struct {
	name             string
	NodeInfoProvider model.NodeInfoProvider
}

func NewNodeDebugInfoProvider(params NodeDebugInfoProviderParams) *NodeDebugInfoProvider {
	return &NodeDebugInfoProvider{
		name:             params.Name,
		NodeInfoProvider: params.NodeInfoProvider,
	}
}

func (r NodeDebugInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	return model.DebugInfo{
		Component: r.name,
		Info:      r.NodeInfoProvider.GetNodeInfo(context.Background()),
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*NodeDebugInfoProvider)(nil)
