package libp2p

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

const ContinuouslyConnectPeersLoopDelaySeconds = 10

// NewHost creates a new libp2p host with some default configuration. It will continuously connect to bootstrap peers
// if they are defined.
func NewHost(port int, opts ...libp2p.Option) (host.Host, error) {
	prvKey, err := config.GetPrivateKey(fmt.Sprintf("private_key.%d", port))
	if err != nil {
		return nil, err
	}

	addrs := []string{
		"/ip4/0.0.0.0/tcp/%d",
		"/ip4/0.0.0.0/udp/%d/quic",
		"/ip4/0.0.0.0/udp/%d/quic-v1",
		"/ip6/::/tcp/%d",
		"/ip6/::/udp/%d/quic",
		"/ip6/::/udp/%d/quic-v1",
	}
	listenAddrs := make([]multiaddr.Multiaddr, 0, len(addrs))
	for _, s := range addrs {
		addr, addrErr := multiaddr.NewMultiaddr(fmt.Sprintf(s, port))
		if addrErr != nil {
			return nil, addrErr
		}
		listenAddrs = append(listenAddrs, addr)
	}

	opts = append(opts, libp2p.ListenAddrs(listenAddrs...))
	opts = append(opts, libp2p.Identity(prvKey))
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("started libp2p host %s listening on: %s", h.ID().String(), h.Addrs())
	return h, err
}

func ConnectToPeersContinuously(ctx context.Context, cm *system.CleanupManager, h host.Host, peers []multiaddr.Multiaddr) error {
	err := ConnectToPeers(ctx, h, peers)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(ContinuouslyConnectPeersLoopDelaySeconds * time.Second)
	ctx, cancelFunction := context.WithCancel(ctx)
	cm.RegisterCallback(func() error {
		cancelFunction()
		return nil
	})
	log.Ctx(ctx).Debug().Msgf("Starting peer reconnection loop every %d seconds", ContinuouslyConnectPeersLoopDelaySeconds)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := ConnectToPeers(ctx, h, peers)
				if err != nil {
					log.Ctx(ctx).Info().Msgf("Error connecting to peers: %s, retrying again in 10 seconds", err)
				}
			case <-ctx.Done():
				log.Ctx(ctx).Debug().Msgf("Reconnect loop stopped")
				return
			}
		}
	}()
	return nil
}

func ConnectToPeers(ctx context.Context, h host.Host, peers []multiaddr.Multiaddr) error {
	errors := []error{}
	for _, peerAddress := range peers {
		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(peerAddress)
		if err != nil {
			errors = append(errors, err)
			log.Ctx(ctx).Warn().Msgf("Error parsing peer address: %s", err)
			continue
		}
		h.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		err = h.Connect(ctx, *info)
		if err != nil {
			errors = append(errors, err)
			log.Ctx(ctx).Warn().Msgf("Error connecting to peer %s: %s, continuing...", info.ID, err)
		} else {
			log.Ctx(ctx).Trace().Msgf("Libp2p transport connected to: %s", peerAddress)
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("libp2p transport had errors connecting to peers: %s", errors)
	}

	return nil
}
