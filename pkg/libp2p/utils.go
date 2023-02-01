package libp2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

func encapsulateP2pAddrs(peerInfo peer.AddrInfo) ([]multiaddr.Multiaddr, error) {
	var allAddrs []multiaddr.Multiaddr
	for _, peerAddrs := range peerInfo.Addrs {
		addr, err := multiaddr.NewMultiaddr("/p2p/" + peerInfo.ID.String())
		if err != nil {
			return nil, err
		}
		allAddrs = append(allAddrs, peerAddrs.Encapsulate(addr))
	}
	return allAddrs, nil
}

func NewHostForTest(ctx context.Context, peers ...host.Host) (host.Host, error) {
	port, err := freeport.GetFreePort()
	if err != nil {
		return nil, err
	}

	h, err := NewHost(port)
	if err != nil {
		return nil, err
	}

	for _, peerHost := range peers {
		if err := connectToPeer(ctx, h, peerHost); err != nil { //nolint:govet
			return nil, err
		}
	}

	return h, err
}

func connectToPeer(ctx context.Context, h host.Host, peer host.Host) error {
	peerAddresses, err := encapsulateP2pAddrs(*host.InfoFromHost(peer))
	if err != nil {
		return err
	}

	log.Ctx(ctx).Debug().
		Stringer("peer", peer.ID()).
		Int("addresses", len(peerAddresses)).
		Msg("Connecting to peer")
	if err := ConnectToPeers(ctx, h, peerAddresses); err != nil { //nolint:govet
		return err
	}

	return err
}
