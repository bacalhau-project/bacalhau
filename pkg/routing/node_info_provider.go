package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type NodeInfoProviderParams struct {
	Host                host.Host
	IdentityService     identify.IDService
	Labels              map[string]string
	ComputeInfoProvider model.ComputeNodeInfoProvider
	BacalhauVersion     model.BuildVersionInfo
}

type NodeInfoProvider struct {
	h                   host.Host
	identityService     identify.IDService
	labels              map[string]string
	computeInfoProvider model.ComputeNodeInfoProvider
	bacalhauVersion     model.BuildVersionInfo
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		h:                   params.Host,
		identityService:     params.IdentityService,
		labels:              params.Labels,
		computeInfoProvider: params.ComputeInfoProvider,
		bacalhauVersion:     params.BacalhauVersion,
	}
}

// RegisterComputeInfoProvider registers a compute info provider with the node info provider.
func (n *NodeInfoProvider) RegisterComputeInfoProvider(provider model.ComputeNodeInfoProvider) {
	n.computeInfoProvider = provider
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) model.NodeInfo {
	res := model.NodeInfo{
		BacalhauVersion: n.bacalhauVersion,
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.identityService.OwnObservedAddrs(),
		},
		Labels: n.labels,
	}
	if n.computeInfoProvider != nil {
		info := n.computeInfoProvider.GetComputeInfo(ctx)
		res.NodeType = model.NodeTypeCompute
		res.ComputeNodeInfo = &info
	}
	return res
}

// compile-time interface check
var _ model.NodeInfoProvider = &NodeInfoProvider{}
