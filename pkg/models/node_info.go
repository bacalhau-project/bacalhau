//go:generate stringer -type=NodeType -trimprefix=NodeType -output=node_info_string.go
package models

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

type NodeInfoProvider interface {
	GetNodeInfo(ctx context.Context) NodeInfo
}

type NodeStateProvider interface {
	GetNodeState(ctx context.Context) NodeState
}

type LabelsProvider interface {
	GetLabels(ctx context.Context) map[string]string
}

type DecoratorNodeInfoProvider interface {
	NodeInfoProvider
	RegisterNodeInfoDecorator(decorator NodeInfoDecorator)
	RegisterLabelProvider(provider LabelsProvider)
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
	NodeID   string            `json:"NodeID"`
	NodeType NodeType          `json:"NodeType"`
	Labels   map[string]string `json:"Labels"`
	// SupportedProtocols indicates which communication protocols this node supports
	SupportedProtocols []Protocol       `json:"SupportedProtocols"`
	ComputeNodeInfo    ComputeNodeInfo  `json:"ComputeNodeInfo,omitempty" yaml:",omitempty"`
	BacalhauVersion    BuildVersionInfo `json:"BacalhauVersion"`
}

// ID returns the node ID
func (n NodeInfo) ID() string {
	return n.NodeID
}

// IsComputeNode returns true if the node is a compute node
func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

// Copy returns a deep copy of the NodeInfo
func (n *NodeInfo) Copy() *NodeInfo {
	if n == nil {
		return nil
	}
	cpy := new(NodeInfo)
	*cpy = *n

	// Deep copy maps
	cpy.Labels = maps.Clone(n.Labels)
	cpy.SupportedProtocols = slices.Clone(n.SupportedProtocols)
	cpy.ComputeNodeInfo = copyOrZero(n.ComputeNodeInfo.Copy())
	cpy.BacalhauVersion = copyOrZero(n.BacalhauVersion.Copy())
	return cpy
}

// HasStaticConfigChanged returns true if the static/configuration aspects of this node
// have changed compared to other. It ignores dynamic operational fields like queue capacity
// and execution counts that change frequently during normal operation.
func (n NodeInfo) HasStaticConfigChanged(other NodeInfo) bool {
	// Define which fields to ignore in the comparison
	opts := []cmp.Option{
		cmpopts.IgnoreFields(ComputeNodeInfo{},
			"QueueUsedCapacity",
			"AvailableCapacity",
			"RunningExecutions",
			"EnqueuedExecutions",
		),
		// Ignore ordering in slices
		cmpopts.SortSlices(func(a, b string) bool { return a < b }),
		cmpopts.SortSlices(func(a, b Protocol) bool { return string(a) < string(b) }),
		cmpopts.SortSlices(func(a, b GPU) bool { return a.Less(b) }), // Sort GPUs by all fields for stable comparison
	}

	return !cmp.Equal(n, other, opts...)
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
	// Address is the network location where this compute node can be reached
	// Format: IPv4 or hostname (e.g., "192.168.1.100" or "node1.example.com")
	Address string `json:"address"`
}

// Copy provides a copy of the allocation and deep copies the job
func (c *ComputeNodeInfo) Copy() *ComputeNodeInfo {
	if c == nil {
		return nil
	}
	cpy := new(ComputeNodeInfo)
	*cpy = *c

	// Deep copy slices
	cpy.ExecutionEngines = slices.Clone(c.ExecutionEngines)
	cpy.Publishers = slices.Clone(c.Publishers)
	cpy.StorageSources = slices.Clone(c.StorageSources)
	cpy.MaxCapacity = copyOrZero(c.MaxCapacity.Copy())
	cpy.QueueUsedCapacity = copyOrZero(c.QueueUsedCapacity.Copy())
	cpy.AvailableCapacity = copyOrZero(c.AvailableCapacity.Copy())
	cpy.MaxJobRequirements = copyOrZero(c.MaxJobRequirements.Copy())
	return cpy
}
