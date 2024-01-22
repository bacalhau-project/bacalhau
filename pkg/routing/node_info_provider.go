package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeInfoProviderParams struct {
	NodeID          string
	LabelsProvider  models.LabelsProvider
	BacalhauVersion models.BuildVersionInfo
}

type NodeInfoProvider struct {
	nodeID             string
	labelsProvider     models.LabelsProvider
	bacalhauVersion    models.BuildVersionInfo
	nodeInfoDecorators []models.NodeInfoDecorator
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		nodeID:             params.NodeID,
		labelsProvider:     params.LabelsProvider,
		bacalhauVersion:    params.BacalhauVersion,
		nodeInfoDecorators: make([]models.NodeInfoDecorator, 0),
	}
}

// RegisterNodeInfoDecorator registers a node info decorator with the node info provider.
func (n *NodeInfoProvider) RegisterNodeInfoDecorator(decorator models.NodeInfoDecorator) {
	n.nodeInfoDecorators = append(n.nodeInfoDecorators, decorator)
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) models.NodeInfo {
	res := models.NodeInfo{
		NodeID:          n.nodeID,
		BacalhauVersion: n.bacalhauVersion,
		Labels:          n.labelsProvider.GetLabels(ctx),
		NodeType:        models.NodeTypeRequester,
	}
	for _, decorator := range n.nodeInfoDecorators {
		res = decorator.DecorateNodeInfo(ctx, res)
	}
	return res
}

// compile-time interface check
var _ models.NodeInfoProvider = &NodeInfoProvider{}
