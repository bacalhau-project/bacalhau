package models

import (
	"context"
)

type BaseNodeInfoProviderParams struct {
	NodeID             string
	LabelsProvider     LabelsProvider
	BacalhauVersion    BuildVersionInfo
	SupportedProtocols []Protocol
}

type BaseNodeInfoProvider struct {
	nodeID             string
	labelsProvider     LabelsProvider
	bacalhauVersion    BuildVersionInfo
	nodeInfoDecorators []NodeInfoDecorator
	supportedProtocol  []Protocol
}

func NewBaseNodeInfoProvider(params BaseNodeInfoProviderParams) *BaseNodeInfoProvider {
	provider := &BaseNodeInfoProvider{
		nodeID:             params.NodeID,
		labelsProvider:     params.LabelsProvider,
		bacalhauVersion:    params.BacalhauVersion,
		nodeInfoDecorators: make([]NodeInfoDecorator, 0),
		supportedProtocol:  params.SupportedProtocols,
	}
	return provider
}

// RegisterNodeInfoDecorator registers a node info decorator with the node info provider.
func (n *BaseNodeInfoProvider) RegisterNodeInfoDecorator(decorator NodeInfoDecorator) {
	n.nodeInfoDecorators = append(n.nodeInfoDecorators, decorator)
}

// RegisterLabelProvider registers a label provider with the node info provider.
func (n *BaseNodeInfoProvider) RegisterLabelProvider(provider LabelsProvider) {
	if n.labelsProvider == nil {
		n.labelsProvider = provider
	} else {
		n.labelsProvider = MergeLabelsInOrder(n.labelsProvider, provider)
	}
}

// GetNodeInfo returns the node info for the node.
func (n *BaseNodeInfoProvider) GetNodeInfo(ctx context.Context) NodeInfo {
	info := NodeInfo{
		NodeID:             n.nodeID,
		BacalhauVersion:    n.bacalhauVersion,
		Labels:             n.labelsProvider.GetLabels(ctx),
		NodeType:           NodeTypeRequester,
		SupportedProtocols: n.supportedProtocol,
	}
	for _, decorator := range n.nodeInfoDecorators {
		info = decorator.DecorateNodeInfo(ctx, info)
	}

	return info
}

// compile-time interface check
var _ NodeInfoProvider = &BaseNodeInfoProvider{}
