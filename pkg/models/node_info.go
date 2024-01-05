//go:generate stringer -type=NodeType -trimprefix=NodeType -output=node_info_string.go
package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
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

type ComputeNodeInfoProvider interface {
	GetComputeInfo(ctx context.Context) ComputeNodeInfo
}

// GetLabels implements LabelsProvider.
func (p mergeProvider) GetLabels(ctx context.Context) map[string]string {
	labels := make(map[string]string)
	for _, provider := range p.providers {
		maps.Copy(labels, provider.GetLabels(ctx))
	}
	return labels
}

type LabelsProvider interface {
	GetLabels(ctx context.Context) map[string]string
}

type mergeProvider struct {
	providers []LabelsProvider
}

func MergeLabelsInOrder(providers ...LabelsProvider) LabelsProvider {
	return mergeProvider{providers: providers}
}

// NodeInfo godoc
//
// @ID			NodeInfo
// @Summary		Information about the node.
// @Description	Information about the node.
// @Tags		Ops
// @Produce		json
// @Success	200	{object}	NodeInfo
// @Failure	500	{object}	string
// @Router		/api/v1/agent/node [get]
type NodeInfo struct {
	PeerInfo        peer.AddrInfo     `json:"PeerInfo" swaggertype:"primitive,integer"`
	NodeType        NodeType          `json:"NodeType"`
	Labels          map[string]string `json:"Labels"`
	ComputeNodeInfo *ComputeNodeInfo  `json:"ComputeNodeInfo,omitempty" yaml:",omitempty"`
	Version         BuildVersionInfo  `json:"Version"`
}

// ID returns the node ID
//
// @ID             NodeInfo_ID
// @Summary        Get node ID.
// @Description    Returns the node ID.
// @Tags           Ops
// @Produce        json
// @Success        200 {string}  string
// @Router         /api/v1/agent/node/{id} [get]
func (n NodeInfo) ID() string {
	return n.PeerInfo.ID.String()
}

func (n NodeInfo) IsComputeNode() bool {
	return n.NodeType == NodeTypeCompute
}

func (n NodeInfo) IsRequesterNode() bool {
	return n.NodeType == NodeTypeRequester
}

type ComputeNodeInfo struct {
	ExecutionEngines   []string  `json:"ExecutionEngines"`
	Publishers         []string  `json:"Publishers"`
	StorageSources     []string  `json:"StorageSources"`
	MaxCapacity        Resources `json:"MaxCapacity"`
	AvailableCapacity  Resources `json:"AvailableCapacity"`
	MaxJobRequirements Resources `json:"MaxJobRequirements"`
	RunningExecutions  int       `json:"RunningExecutions"`
	EnqueuedExecutions int       `json:"EnqueuedExecutions"`
}
