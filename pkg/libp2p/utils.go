package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

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

// upgradeAddress accepts a non p2p multiaddress that denotes a http or https connection
// and attempts to use it to call the requester node to ask for it's peer id.  If
// successful it will then convert the http(s) multiaddress into a valid p2p multiaddress
// and return it for use by the caller.
func upgradeAddress(ctx context.Context, address multiaddr.Multiaddr) (multiaddr.Multiaddr, error) {
	type PeerInfo struct {
		ID string `json:"ID"`
	}

	type PeerInfoResponse struct {
		Peerinfo PeerInfo `json:"PeerInfo"`
	}

	parts := strings.Split(address.String()[1:], "/")
	ipAddr, port, scheme := parts[1], parts[3], parts[4]
	url := fmt.Sprintf("%s://%s:%s/api/v1/agent/node", scheme, ipAddr, port)

	client := http.Client{Timeout: time.Second * 2}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return address, err
	}

	res, err := client.Do(req)
	if err != nil {
		return address, err
	}

	defer func() {
		if res.Body != nil {
			res.Body.Close()
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return address, err
	}

	response := PeerInfoResponse{}
	if err = json.Unmarshal(body, &response); err != nil {
		return address, nil
	}

	if response.Peerinfo.ID != "" {
		if addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/%s", scheme)); err != nil {
			return address, err
		} else {
			address = address.Decapsulate(addr)
		}

		if p2pAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", response.Peerinfo.ID)); err != nil {
			return address, nil
		} else {
			address = address.Encapsulate(p2pAddr)
		}
	}

	return address, nil
}
