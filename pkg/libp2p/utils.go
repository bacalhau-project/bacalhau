package libp2p

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
)

func ExtractAddrInfoFromHost(host host.Host) peer.AddrInfo {
	return peer.AddrInfo{
		ID:    host.ID(),
		Addrs: host.Addrs(),
	}
}

func EncapsulateP2pAddrsFromHosts(hosts ...host.Host) ([]multiaddr.Multiaddr, error) {
	var allAddrs []multiaddr.Multiaddr
	for _, h := range hosts {
		peerInfo := ExtractAddrInfoFromHost(h)
		encapsulatedAddrs, err := EncapsulateP2pAddrs(peerInfo)
		if err != nil {
			return nil, err
		}
		allAddrs = append(allAddrs, encapsulatedAddrs...)
	}
	return allAddrs, nil
}

func EncapsulateP2pAddrs(peersInfo ...peer.AddrInfo) ([]multiaddr.Multiaddr, error) {
	var allAddrs []multiaddr.Multiaddr
	for _, peerInfo := range peersInfo {
		for _, peerAddrs := range peerInfo.Addrs {
			addr, err := multiaddr.NewMultiaddr("/p2p/" + peerInfo.ID.String())
			if err != nil {
				return nil, err
			}
			allAddrs = append(allAddrs, peerAddrs.Encapsulate(addr))
		}
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

	libp2pPeer, err := EncapsulateP2pAddrsFromHosts(peers...)
	if err != nil {
		return nil, err
	}
	if len(libp2pPeer) > 0 {
		err = ConnectToPeers(ctx, h, libp2pPeer)
	}

	return h, err
}
