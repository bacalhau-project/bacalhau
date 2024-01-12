package peermgr

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/libp2p/go-libp2p/core/host"
	net "github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/slices"
)

type PeerMgr struct {
	// the libp2p host
	h host.Host

	// peers we are instructing peermrg to maintain connections with.
	bootstrappers []multiaddr.Multiaddr

	// signals service is currently bootstrapping to bootstrap peers.
	bootstrapping chan struct{}

	// peers we are currently connected to.
	peersMu sync.Mutex
	peers   map[peer.ID]struct{}

	notifee *net.NotifyBundle

	// signals service exit
	done chan struct{}

	cfg config
}

func New(h host.Host, bootstrap []multiaddr.Multiaddr, opts ...Option) (*PeerMgr, error) {
	cfg := &config{
		runInterval:      time.Second * 5,
		bootstrapTimeout: time.Second * 30,
		// default is a real clock
		clock: clock.New(),
	}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, err
		}
	}

	pm := &PeerMgr{
		h:             h,
		bootstrappers: bootstrap,
		peers:         make(map[peer.ID]struct{}),
		done:          make(chan struct{}),
		bootstrapping: make(chan struct{}, 1),
		cfg:           *cfg,
	}

	// register notification for peer dis/connection, used to maintain peers set.
	pm.notifee = &net.NotifyBundle{
		// triggered whenever we disconnect from a peer
		DisconnectedF: func(_ net.Network, conn net.Conn) {
			pm.handleDisconnect(conn.RemotePeer())
		},
		// triggered whenever we connect to a peer
		ConnectedF: func(_ net.Network, conn net.Conn) {
			pm.handleConnect(conn.RemotePeer())
		},
	}

	h.Network().Notify(pm.notifee)

	return pm, nil
}

func (m *PeerMgr) PeerCount() int {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()
	return len(m.peers)
}

func (m *PeerMgr) Start(ctx context.Context) {
	log.Ctx(ctx).Info().Msg("starting peermgr")
	go m.Run(ctx)
}

func (m *PeerMgr) Stop(ctx context.Context) {
	log.Ctx(ctx).Info().Msg("stopping peermgr")
	close(m.done)
}

func (m *PeerMgr) Run(ctx context.Context) {
	// eagerly do bootstrapping at startup, after initial run look interval
	m.doBootstrap(ctx)
	tick := m.cfg.clock.Ticker(m.cfg.runInterval)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			current := m.PeerCount()
			threshold := len(m.bootstrappers)
			if current < threshold {
				log.Ctx(ctx).Info().Int("current", current).Int("threshold", threshold).
					Msg("connected bootstrap peers below threshold, running bootstrap routine")
				m.bootstrapPeers(ctx)
			}
		case <-m.done:
			log.Ctx(ctx).Info().Msg("exiting peermgr")
			return
		}
	}
}

func (m *PeerMgr) handleDisconnect(p peer.ID) {
	disconnected := false

	if m.h.Network().Connectedness(p) == net.NotConnected {
		m.peersMu.Lock()
		// if we were connected to this peer remove it from our set of connected peers.
		_, disconnected = m.peers[p]
		if disconnected {
			delete(m.peers, p)
		}
		m.peersMu.Unlock()
	}

	if disconnected {
		log.Info().Str("peer", p.String()).Msg("disconnected from peer")
	}
}

func (m *PeerMgr) handleConnect(p peer.ID) {
	if m.h.Network().Connectedness(p) == net.Connected {
		// add this peer to the set of connected peers.
		m.peersMu.Lock()
		m.peers[p] = struct{}{}
		m.peersMu.Unlock()
		log.Info().Str("peer", p.String()).Msg("connected to peer")
	}
}

func (m *PeerMgr) bootstrapPeers(ctx context.Context) {
	select {
	case m.bootstrapping <- struct{}{}:
	default:
		return
	}

	go func(ctx context.Context) {
		bsctx, cancel := context.WithTimeout(ctx, m.cfg.bootstrapTimeout)
		defer cancel()

		m.doBootstrap(bsctx)

		<-m.bootstrapping
	}(ctx)
}

func (m *PeerMgr) doBootstrap(ctx context.Context) {
	wg := sync.WaitGroup{}
	for _, bsp := range m.bootstrappers {
		wg.Add(1)
		go func(addr multiaddr.Multiaddr) {
			defer wg.Done()
			if err := m.connectAddress(ctx, addr); err != nil {
				log.Ctx(ctx).Warn().Err(err).Stringer("address", addr).
					Msgf("failed to connect to bootstrap peer")
			}
		}(bsp)
	}
	wg.Wait()
}

func (m *PeerMgr) connectAddress(ctx context.Context, addr multiaddr.Multiaddr) error {
	// Lookup whether the address we've been given might be a http(s) multiaddress and if so then
	// we will attempt to fetch the peer id from the remote address, and encapsulate a new address
	// from that data.
	upgraded := false
	isHTTP := func(p multiaddr.Protocol) bool { return p.Name == "http" || p.Name == "https" }
	if slices.ContainsFunc(addr.Protocols(), isHTTP) {
		upgrade, err := upgradeAddress(ctx, addr)
		if err != nil {
			return fmt.Errorf("attempting to upgrade multi-address %s to peer address: %w", addr, err)
		}
		log.Ctx(ctx).Info().Stringer("upgrade", upgrade).Stringer("original", addr).Msg("upgraded address")
		addr = upgrade
		upgraded = true
	}

	info, err := peer.AddrInfoFromP2pAddr(addr)
	if err != nil {
		return fmt.Errorf("parsing peer address %s: %w", addr, err)
	}
	if upgraded {
		log.Ctx(ctx).Info().Stringer("address", addr).Stringer("peer", info).
			Msg("upgraded multiaddress to peer address")
	}

	if err := m.h.Connect(ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to peer %s: %w", info, err)
	}
	return nil
}

// upgradeAddress accepts a non p2p multiaddress that denotes a http or https connection
// and attempts to use it to call the requester node to ask for it's peer id.  If
// successful it will then convert the http(s) multiaddress into a valid p2p multiaddress
// and return it for use by the caller.
func upgradeAddress(ctx context.Context, address multiaddr.Multiaddr) (multiaddr.Multiaddr, error) {
	type PeerInfo struct {
		ID    string   `json:"ID"`
		Addrs []string `json:"Addrs"`
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
		// Strip the http(s) from the end of the address
		if addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/%s", scheme)); err != nil {
			return address, err
		} else {
			address = address.Decapsulate(addr)
		}

		// Strip the TCP/port section from the address
		if tcp, err := multiaddr.NewMultiaddr(fmt.Sprintf("/tcp/%s", port)); err != nil {
			return address, err
		} else {
			address = address.Decapsulate(tcp)
		}

		// TODO: Fixed to the most common port for now, until we are able to find the valid
		// listen port but should not rely on this.
		if tcp, err := multiaddr.NewMultiaddr("/tcp/1235"); err != nil {
			return address, err
		} else {
			address = address.Encapsulate(tcp)
		}

		// Add the p2p/peerid component to the address
		if p2pAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/p2p/%s", response.Peerinfo.ID)); err != nil {
			return address, nil
		} else {
			address = address.Encapsulate(p2pAddr)
		}
	}

	return address, nil
}
