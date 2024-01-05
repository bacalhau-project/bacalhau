package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type NodeInfoProviderParams struct {
	Host                host.Host
	IdentityService     identify.IDService
	Labels              map[string]string
	ComputeInfoProvider models.ComputeNodeInfoProvider
	Version             models.BuildVersionInfo
}

type NodeInfoProvider struct {
	h                   host.Host
	identityService     identify.IDService
	labels              map[string]string
	computeInfoProvider models.ComputeNodeInfoProvider
	version             models.BuildVersionInfo
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		h:                   params.Host,
		identityService:     params.IdentityService,
		labels:              params.Labels,
		computeInfoProvider: params.ComputeInfoProvider,
		version:             params.Version,
	}
}

// RegisterComputeInfoProvider registers a compute info provider with the node info provider.
func (n *NodeInfoProvider) RegisterComputeInfoProvider(provider models.ComputeNodeInfoProvider) {
	n.computeInfoProvider = provider
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) models.NodeInfo {
	res := models.NodeInfo{
		Version: n.version,
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.identityService.OwnObservedAddrs(),
		},
		Labels:   n.labels,
		NodeType: models.NodeTypeRequester,
	}
	if n.computeInfoProvider != nil {
		info := n.computeInfoProvider.GetComputeInfo(ctx)
		res.NodeType = models.NodeTypeCompute
		res.ComputeNodeInfo = &info
	}
	return res
}

// compile-time interface check
var _ models.NodeInfoProvider = &NodeInfoProvider{}
