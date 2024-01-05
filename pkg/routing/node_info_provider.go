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
	LabelsProvider      models.LabelsProvider
	ComputeInfoProvider models.ComputeNodeInfoProvider
	Version             models.BuildVersionInfo
}

type NodeInfoProvider struct {
	h                   host.Host
	identityService     identify.IDService
	labelsProvider      models.LabelsProvider
	computeInfoProvider models.ComputeNodeInfoProvider
	version             models.BuildVersionInfo
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		h:                   params.Host,
		identityService:     params.IdentityService,
		labelsProvider:      params.LabelsProvider,
		computeInfoProvider: params.ComputeInfoProvider,
		version:             params.Version,
	}
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) models.NodeInfo {
	res := models.NodeInfo{
		Version: n.version,
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.identityService.OwnObservedAddrs(),
		},
		Labels:   n.labelsProvider.GetLabels(ctx),
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
