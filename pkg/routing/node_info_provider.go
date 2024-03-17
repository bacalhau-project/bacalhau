package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeInfoProviderParams struct {
	NodeID              string
	LabelsProvider      models.LabelsProvider
	BacalhauVersion     models.BuildVersionInfo
	DefaultNodeApproval models.NodeApproval
}

type NodeInfoProvider struct {
	nodeID              string
	labelsProvider      models.LabelsProvider
	bacalhauVersion     models.BuildVersionInfo
	nodeInfoDecorators  []models.NodeInfoDecorator
	defaultNodeApproval models.NodeApproval
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	provider := &NodeInfoProvider{
		nodeID:              params.NodeID,
		labelsProvider:      params.LabelsProvider,
		bacalhauVersion:     params.BacalhauVersion,
		nodeInfoDecorators:  make([]models.NodeInfoDecorator, 0),
		defaultNodeApproval: params.DefaultNodeApproval,
	}

	// If we were not given a default approval, we default to PENDING
	if !provider.defaultNodeApproval.IsValid() {
		provider.defaultNodeApproval = models.NodeApprovals.PENDING
	}

	return provider
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
		Approval:        n.defaultNodeApproval,
	}
	for _, decorator := range n.nodeInfoDecorators {
		res = decorator.DecorateNodeInfo(ctx, res)
	}

	if !res.Approval.IsValid() {
		res.Approval = models.NodeApprovals.PENDING
	}

	return res
}

// compile-time interface check
var _ models.NodeInfoProvider = &NodeInfoProvider{}
