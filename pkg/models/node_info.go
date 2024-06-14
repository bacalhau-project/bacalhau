//go:generate stringer -type=NodeType -trimprefix=NodeType -output=node_info_string.go
package models

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"
)

type NodeType int

const (
	nodeTypeUndefined NodeType = iota
	NodeTypeRequester
	NodeTypeCompute
)

func ParseNodeType(s string) (NodeType, error) {
	for typ := NodeTypeRequester; typ <= NodeTypeCompute; typ++ {
		if strings.EqualFold(typ.String(), strings.TrimSpace(s)) {
			return typ, nil
		}
	}

	return nodeTypeUndefined, fmt.Errorf("invalid node type: %s", s)
}

func (e NodeType) MarshalText() ([]byte, error) {
	return []byte(e.String()), nil
}

func (e *NodeType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*e, err = ParseNodeType(name)
	return
}

type NodeStateProvider interface {
	GetNodeState(ctx context.Context) NodeState
}

type LabelsProvider interface {
	GetLabels(ctx context.Context) map[string]string
}

type mergeProvider struct {
	providers []LabelsProvider
}

// GetLabels implements LabelsProvider.
func (p mergeProvider) GetLabels(ctx context.Context) map[string]string {
	labels := make(map[string]string)
	for _, provider := range p.providers {
		maps.Copy(labels, provider.GetLabels(ctx))
	}
	return labels
}

func MergeLabelsInOrder(providers ...LabelsProvider) LabelsProvider {
	return mergeProvider{providers: providers}
}

type NodeInfoDecorator interface {
	DecorateNodeInfo(ctx context.Context, nodeInfo NodeInfo) NodeInfo
}

// NoopNodeInfoDecorator is a decorator that does nothing
type NoopNodeInfoDecorator struct{}

func (n NoopNodeInfoDecorator) DecorateNodeInfo(ctx context.Context, nodeInfo NodeInfo) NodeInfo {
	return nodeInfo
}

// NodeInfo contains metadata about a node on the network. Compute nodes share their NodeInfo with Requester nodes
// to further its view of the networks conditions. ComputeNodeInfo is non-nil iff the NodeType is NodeTypeCompute.
// TODO(walid): add Validate() method to NodeInfo and make sure it is called in all the places where it is initialized
type NodeInfo struct {
	// TODO replace all access on this field with the `ID()` method
	NodeID          string            `json:"NodeID"`
	NodeType        NodeType          `json:"NodeType"`
	Labels          map[string]string `json:"Labels"`
	ComputeNodeInfo *ComputeNodeInfo  `json:"ComputeNodeInfo,omitempty" yaml:",omitempty"`
	BacalhauVersion BuildVersionInfo  `json:"BacalhauVersion"`
}

// ID returns the node ID
func (n NodeInfo) ID() string {
	return n.NodeID
}

// IsComputeNode returns true if the node is a compute node
func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

// ComputeNodeInfo contains metadata about the current state and abilities of a compute node. Compute Nodes share
// this state with Requester nodes by including it in the NodeInfo they share across the network.
type ComputeNodeInfo struct {
	ExecutionEngines   []string  `json:"ExecutionEngines"`
	Publishers         []string  `json:"Publishers"`
	StorageSources     []string  `json:"StorageSources"`
	MaxCapacity        Resources `json:"MaxCapacity"`
	QueueUsedCapacity  Resources `json:"QueueCapacity"`
	AvailableCapacity  Resources `json:"AvailableCapacity"`
	MaxJobRequirements Resources `json:"MaxJobRequirements"`
	RunningExecutions  int       `json:"RunningExecutions"`
	EnqueuedExecutions int       `json:"EnqueuedExecutions"`
}
