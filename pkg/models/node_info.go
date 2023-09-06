//go:generate stringer -type=NodeType -trimprefix=NodeType -output=node_info_string.go
package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
)

type NodeType int

const (
	NodeTypeRequester NodeType = iota
	NodeTypeCompute
)

func ParseNodeType(s string) (NodeType, error) {
	for typ := NodeTypeRequester; typ <= NodeTypeCompute; typ++ {
		if strings.EqualFold(typ.String(), strings.TrimSpace(s)) {
			return typ, nil
		}
	}

	return NodeTypeCompute, fmt.Errorf("invalid node type: %s", s)
}

type NodeInfoProvider interface {
	GetNodeInfo(ctx context.Context) NodeInfo
}

type ComputeNodeInfoProvider interface {
	GetComputeInfo(ctx context.Context) ComputeNodeInfo
}

type NodeInfo struct {
	BacalhauVersion BuildVersionInfo
	PeerInfo        peer.AddrInfo
	NodeType        NodeType
	Labels          map[string]string
	ComputeNodeInfo *ComputeNodeInfo
}

// IsComputeNode returns true if the node is a compute node
func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

type ComputeNodeInfo struct {
	ExecutionEngines   []string
	Publishers         []string
	StorageSources     []string
	MaxCapacity        Resources
	AvailableCapacity  Resources
	MaxJobRequirements Resources
	RunningExecutions  int
	EnqueuedExecutions int
}
