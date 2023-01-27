package routing

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type NodeInfoProviderParams struct {
	Host                host.Host
	IdentityService     identify.IDService
	Labels              map[string]string
	ComputeInfoProvider model.ComputeNodeInfoProvider
}

type NodeInfoProvider struct {
	h                   host.Host
	identityService     identify.IDService
	labels              map[string]string
	computeInfoProvider model.ComputeNodeInfoProvider
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		h:                   params.Host,
		identityService:     params.IdentityService,
		labels:              params.Labels,
		computeInfoProvider: params.ComputeInfoProvider,
	}
}

// RegisterComputeInfoProvider registers a compute info provider with the node info provider.
func (n *NodeInfoProvider) RegisterComputeInfoProvider(provider model.ComputeNodeInfoProvider) {
	n.computeInfoProvider = provider
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) model.NodeInfo {
	res := model.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.identityService.OwnObservedAddrs(),
		},
		Labels: n.labels,
	}
	if n.computeInfoProvider != nil {
		res.NodeType = model.NodeTypeCompute
		res.ComputeNodeInfo = n.computeInfoProvider.GetComputeInfo(ctx)
	}
	return res
}

// compile-time interface check
var _ model.NodeInfoProvider = &NodeInfoProvider{}
