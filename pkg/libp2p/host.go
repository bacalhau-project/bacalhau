package libp2p

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/generic"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

const continuouslyConnectPeersLoopDelay = 10 * time.Second

// NewHost creates a new libp2p host with some default configuration. It will continuously connect to bootstrap peers
// if they are defined.
func NewHost(port int, opts ...libp2p.Option) (host.Host, error) {
	prvKey, err := config.GetPrivateKey(fmt.Sprintf("private_key.%d", port))
	if err != nil {
		return nil, err
	}

	addrs := []string{
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", port),
		fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic-v1", port),
		fmt.Sprintf("/ip6/::/tcp/%d", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic", port),
		fmt.Sprintf("/ip6/::/udp/%d/quic-v1", port),
	}

	preferredAddress := config.PreferredAddress()
	if preferredAddress != "" {
		newAddress := fmt.Sprintf("/ip4/%s/tcp/0", preferredAddress)
		addrs = append(addrs, newAddress)
	}

	opts = append(opts, libp2p.ListenAddrStrings(addrs...))
	opts = append(opts, libp2p.Identity(prvKey))
	h, err := libp2p.New(opts...)
	if err != nil {
		return nil, err
	}

	p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + h.ID().String())
	if err != nil {
		return nil, err
	}
	addresses := generic.Map[multiaddr.Multiaddr, fmt.Stringer](h.Addrs(), func(m multiaddr.Multiaddr) fmt.Stringer {
		return m
	})
	p2pAddresses := generic.Map[multiaddr.Multiaddr, fmt.Stringer](h.Addrs(), func(m multiaddr.Multiaddr) fmt.Stringer {
		return m.Encapsulate(p2pAddr)
	})

	log.Debug().
		Stringers("listening-addresses", addresses).
		Stringers("p2p-addresses", p2pAddresses).
		Stringer("host-id", h.ID()).
		Msgf("started libp2p host")

	return h, err
}

func ConnectToPeersContinuously(ctx context.Context, cm *system.CleanupManager, h host.Host, peers []multiaddr.Multiaddr) error {
	return ConnectToPeersContinuouslyWithRetryDuration(ctx, cm, h, peers, continuouslyConnectPeersLoopDelay)
}

func ConnectToPeersContinuouslyWithRetryDuration(
	ctx context.Context,
	cm *system.CleanupManager,
	h host.Host,
	peers []multiaddr.Multiaddr,
	tickDuration time.Duration,
) error {
	if err := connectToPeers(ctx, h, peers); err != nil {
		return err
	}
	ticker := time.NewTicker(tickDuration)
	ctx, cancel := context.WithCancel(ctx)
	cm.RegisterCallback(func() error {
		cancel()
		return nil
	})
	log.Ctx(ctx).Debug().Stringer("tick", tickDuration).Msg("Starting peer reconnection loop")
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := connectToPeers(ctx, h, peers); err != nil {
					log.Ctx(ctx).Info().
						Err(err).
						Stringer("tick", tickDuration).
						Msg("Error connecting to peers")
				}
			case <-ctx.Done():
				log.Ctx(ctx).Debug().Msg("Reconnect loop stopped")
				return
			}
		}
	}()
	return nil
}

func connectToPeers(ctx context.Context, h host.Host, peers []multiaddr.Multiaddr) error {
	// The call to `Connect` will "block until a connection is open, or an error is returned". This could mean the
	// request to open a connection is never seen if the peer has only just started up. The default dial timeout
	// is 60 seconds.
	ctx = network.WithDialPeerTimeout(ctx, 5*time.Second) //nolint:gomnd

	var errors []error
	grouped := map[peer.ID][]multiaddr.Multiaddr{}

	// Group up the peers by ID, so we only connect to a peer once rather than multiple times
	for _, peerAddress := range peers {
		info, err := peer.AddrInfoFromP2pAddr(peerAddress)
		if err != nil {
			errors = append(errors, err)
			log.Ctx(ctx).Warn().Err(err).Msg("Error parsing peer address")
			continue
		}

		grouped[info.ID] = append(grouped[info.ID], info.Addrs...)
	}

	for id, addresses := range grouped {
		h.Peerstore().AddAddrs(id, addresses, peerstore.PermanentAddrTTL)
		err := h.Connect(ctx, peer.AddrInfo{
			ID:    id,
			Addrs: addresses,
		})
		if err != nil {
			errors = append(errors, err)
			log.Ctx(ctx).Warn().
				Err(err).
				Stringers("addresses", logger.ToSliceStringer(addresses, multiAddressToString)).
				Stringer("peer", id).
				Msg("Error connecting to peer, continuing...")
		} else {
			log.Ctx(ctx).Trace().
				Stringers("addresses", logger.ToSliceStringer(addresses, multiAddressToString)).
				Stringer("peer", id).
				Msg("Libp2p transport connected to peer")
		}
	}
	if len(errors) > 0 {
		return fmt.Errorf("libp2p transport had errors connecting to peers: %s", errors)
	}

	return nil
}

func multiAddressToString(t multiaddr.Multiaddr) string {
	return t.String()
}
