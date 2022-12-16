package requester

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

type IdentityNodeDiscovererParams struct {
	Host host.Host
}

type IdentityNodeDiscoverer struct {
	host host.Host
}

func NewIdentityNodeDiscoverer(params IdentityNodeDiscovererParams) *IdentityNodeDiscoverer {
	return &IdentityNodeDiscoverer{
		host: params.Host,
	}
}

func (d *IdentityNodeDiscoverer) FindNodes(ctx context.Context, job model.Job) ([]peer.ID, error) {
	peers := []peer.ID{}

	// check local protocols in case the current node is also a compute node
	// peerstore doesn't seem to hold protocols of the current node
	for _, protocol := range d.host.Mux().Protocols() {
		if protocol == bprotocol.AskForBidProtocolID {
			peers = append(peers, d.host.ID())
		}
	}

	for _, peerID := range d.host.Peerstore().Peers() {
		if peerID == d.host.ID() {
			continue
		}
		supportedProtocols, err := d.host.Peerstore().SupportsProtocols(peerID, bprotocol.AskForBidProtocolID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to get supported protocols for peer %s", peerID)
			continue
		}
		if len(supportedProtocols) > 0 {
			peers = append(peers, peerID)
		}
	}
	return peers, nil
}

// compile time check that IdentityNodeDiscoverer implements NodeDiscoverer
var _ NodeDiscoverer = (*IdentityNodeDiscoverer)(nil)
