package libp2p

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type PeerInfoDecoratorParams struct {
	Host            host.Host
	IdentityService identify.IDService
}

type PeerInfoDecorator struct {
	host            host.Host
	identityService identify.IDService
}

func NewPeerInfoDecorator(params PeerInfoDecoratorParams) *PeerInfoDecorator {
	return &PeerInfoDecorator{
		host:            params.Host,
		identityService: params.IdentityService,
	}
}

func (l *PeerInfoDecorator) DecorateNodeInfo(ctx context.Context, nodeInfo models.NodeInfo) models.NodeInfo {
	nodeInfo.PeerInfo = &peer.AddrInfo{
		ID:    l.host.ID(),
		Addrs: l.identityService.OwnObservedAddrs(),
	}
	return nodeInfo
}

// compile-time check whether the PeerInfoDecorator implements the PeerInfoDecorator interface.
var _ models.NodeInfoDecorator = (*PeerInfoDecorator)(nil)
