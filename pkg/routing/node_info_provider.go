package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeStateProviderParams struct {
	NodeID              string
	LabelsProvider      models.LabelsProvider
	BacalhauVersion     models.BuildVersionInfo
	DefaultNodeApproval models.NodeApproval
}

type NodeStateProvider struct {
	nodeID              string
	labelsProvider      models.LabelsProvider
	bacalhauVersion     models.BuildVersionInfo
	nodeInfoDecorators  []models.NodeInfoDecorator
	defaultNodeApproval models.NodeApproval
}

func NewNodeStateProvider(params NodeStateProviderParams) *NodeStateProvider {
	provider := &NodeStateProvider{
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
func (n *NodeStateProvider) RegisterNodeInfoDecorator(decorator models.NodeInfoDecorator) {
	n.nodeInfoDecorators = append(n.nodeInfoDecorators, decorator)
}

func (n *NodeStateProvider) GetNodeState(ctx context.Context) models.NodeState {
	info := models.NodeInfo{
		NodeID:          n.nodeID,
		BacalhauVersion: n.bacalhauVersion,
		Labels:          n.labelsProvider.GetLabels(ctx),
		NodeType:        models.NodeTypeRequester,
	}
	for _, decorator := range n.nodeInfoDecorators {
		info = decorator.DecorateNodeInfo(ctx, info)
	}

	state := models.NodeState{
		Info:     info,
		Approval: n.defaultNodeApproval,
		// TODO what is the nodes state here?
		// Liveness: models.NodeLiveness{},
	}

	if !state.Approval.IsValid() {
		state.Approval = models.NodeApprovals.PENDING
	}

	return state
}

// compile-time interface check
var _ models.NodeStateProvider = &NodeStateProvider{}
