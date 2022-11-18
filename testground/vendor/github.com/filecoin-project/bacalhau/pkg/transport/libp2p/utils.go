package libp2p

import (
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
)

// Encapsulates multiaddrs with p2p protocol
func EncapsulateP2PAddrs(hostID peer.ID, addrs []multiaddr.Multiaddr) ([]multiaddr.Multiaddr, error) {
	var p2pAddrs []multiaddr.Multiaddr
	for _, addr := range addrs {
		p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + hostID.String())
		if err != nil {
			return nil, err
		}
		p2pAddrs = append(p2pAddrs, addr.Encapsulate(p2pAddr))
	}
	return p2pAddrs, nil
}
