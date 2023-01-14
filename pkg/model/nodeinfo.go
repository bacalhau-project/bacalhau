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
	PeerInfo        peer.AddrInfo   `json:"peerInfo"`
	NodeType        NodeType        `json:"nodeType"`
	ComputeNodeInfo ComputeNodeInfo `json:"computeNodeInfo"`
}

// IsComputeNode returns true if the node is a compute node
func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

type ComputeNodeInfo struct {
	ExecutionEngines   []Engine          `json:"executionEngines"`
	MaxCapacity        ResourceUsageData `json:"maxCapacity"`
	AvailableCapacity  ResourceUsageData `json:"availableCapacity"`
	MaxJobRequirements ResourceUsageData `json:"maxJobRequirements"`
}
