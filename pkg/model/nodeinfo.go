package model

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

type NodeType int

const (
	nodeTypeUnknown NodeType = iota
	NodeTypeCompute
)

type NodeInfo struct {
	PeerInfo        peer.AddrInfo   `json:"PeerInfo"`
	NodeType        NodeType        `json:"NodeType"`
	ComputeNodeInfo ComputeNodeInfo `json:"ComputeNodeInfo"`
}

// IsComputeNode returns true if the node is a compute node
func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

type ComputeNodeInfo struct {
	ExecutionEngines   []Engine          `json:"ExecutionEngines"`
	MaxCapacity        ResourceUsageData `json:"MaxCapacity"`
	AvailableCapacity  ResourceUsageData `json:"AvailableCapacity"`
	MaxJobRequirements ResourceUsageData `json:"MaxJobRequirements"`
	RunningExecutions  int               `json:"RunningExecutions"`
	EnqueuedExecutions int               `json:"EnqueuedExecutions"`
}
