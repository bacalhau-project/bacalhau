package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeStateProviderParams struct {
	NodeID              string
	LabelsProvider      models.LabelsProvider
	BacalhauVersion     models.BuildVersionInfo
	DefaultNodeApproval models.NodeMembershipState
	SupportedProtocols  []models.Protocol
}

type NodeStateProvider struct {
	nodeID              string
	labelsProvider      models.LabelsProvider
	bacalhauVersion     models.BuildVersionInfo
	nodeInfoDecorators  []models.NodeInfoDecorator
	defaultNodeApproval models.NodeMembershipState
	supportedProtocol   []models.Protocol
}

func NewNodeStateProvider(params NodeStateProviderParams) *NodeStateProvider {
	provider := &NodeStateProvider{
		nodeID:              params.NodeID,
		labelsProvider:      params.LabelsProvider,
		bacalhauVersion:     params.BacalhauVersion,
		nodeInfoDecorators:  make([]models.NodeInfoDecorator, 0),
		defaultNodeApproval: params.DefaultNodeApproval,
		supportedProtocol:   params.SupportedProtocols,
	}

	// If we were not given a default approval, we default to PENDING
	if !provider.defaultNodeApproval.IsValid() {
		provider.defaultNodeApproval = models.NodeMembership.PENDING
	}

	return provider
}

// RegisterNodeInfoDecorator registers a node info decorator with the node info provider.
func (n *NodeStateProvider) RegisterNodeInfoDecorator(decorator models.NodeInfoDecorator) {
	n.nodeInfoDecorators = append(n.nodeInfoDecorators, decorator)
}

func (n *NodeStateProvider) GetNodeState(ctx context.Context) models.NodeState {
	info := models.NodeInfo{
		NodeID:             n.nodeID,
		BacalhauVersion:    n.bacalhauVersion,
		Labels:             n.labelsProvider.GetLabels(ctx),
		NodeType:           models.NodeTypeRequester,
		SupportedProtocols: n.supportedProtocol,
	}
	for _, decorator := range n.nodeInfoDecorators {
		info = decorator.DecorateNodeInfo(ctx, info)
	}

	state := models.NodeState{
		Info:       info,
		Membership: n.defaultNodeApproval,
		// NB(forrest): we are returning NodeState about ourselves (the Requester)
		// the concept of a disconnected requester node could only exist from the
		// perspective of a ComputeNode or another RequesterNode.
		// We don't support multiple requester nodes nor querying the state of one from a Compute node. (yet)
		// So we allways say we are connected here.

		// This is all pretty funky and my comment here will hopefully become outdates at some-point and need adjusting,
		// but for now: "you can tell the requester node is connected because of the way it is".
		Connection: models.NodeStates.CONNECTED,
	}

	if !state.Membership.IsValid() {
		state.Membership = models.NodeMembership.PENDING
	}

	return state
}

// compile-time interface check
var _ models.NodeStateProvider = &NodeStateProvider{}
