package libp2p

import (
	"context"
	"crypto/rand"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

const DefaultKeySize = 2048

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

	privKey, err := GeneratePrivateKey(DefaultKeySize)
	if err != nil {
		return nil, err
	}
	h, err := NewHost(port, privKey)
	if err != nil {
		return nil, err
	}

	for _, peerHost := range peers {
		if err := ConnectToPeer(ctx, h, peerHost); err != nil {
			return nil, err
		}
	}

	return h, err
}

func ConnectToPeer(ctx context.Context, h host.Host, peer host.Host) error {
	peerAddresses, err := encapsulateP2pAddrs(*host.InfoFromHost(peer))
	if err != nil {
		return err
	}

	log.Ctx(ctx).Debug().
		Stringer("peer", peer.ID()).
		Int("addresses", len(peerAddresses)).
		Msg("Connecting to peer")
	if err := connectToPeers(ctx, h, peerAddresses); err != nil {
		return err
	}

	return err
}

func GeneratePrivateKey(numBits int) (crypto.PrivKey, error) {
	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, numBits, rand.Reader)
	if err != nil {
		return nil, err
	}
	return prvKey, nil
}
