package discovery

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/stretchr/testify/suite"
)

type IdentityNodeDiscovererSuite struct {
	suite.Suite
	discoverer *IdentityNodeDiscoverer
	node1      host.Host
	node2      host.Host
	random3    host.Host
}

func (s *IdentityNodeDiscovererSuite) SetupSuite() {
	// create 3 nodes network all connected to the first one as a bootstrap node,
	// where only the first two nodes are implementing bprotcol
	ctx := context.Background()
	var err error
	s.node1, err = libp2p.NewHostForTest(ctx)
	s.NoError(err)
	s.node2, err = libp2p.NewHostForTest(ctx)
	s.NoError(err)
	s.random3, err = libp2p.NewHostForTest(ctx)
	s.NoError(err)

	// implement bprotocol
	s.node1.SetStreamHandler(bprotocol.AskForBidProtocolID, func(s network.Stream) {})
	s.node2.SetStreamHandler(bprotocol.AskForBidProtocolID, func(s network.Stream) {})

	// connect all nodes after implementing bprotocol to share the info in the initial handshake
	s.NoError(s.node2.Connect(ctx, *host.InfoFromHost(s.node1)))
	s.NoError(s.random3.Connect(ctx, *host.InfoFromHost(s.node1)))

	s.discoverer = NewIdentityNodeDiscoverer(IdentityNodeDiscovererParams{
		Host: s.node1,
	})
}

func (s *IdentityNodeDiscovererSuite) TearDownSuite() {
	s.NoError(s.node1.Close())
	s.NoError(s.node2.Close())
	s.NoError(s.random3.Close())
}

func TestIdentityNodeDiscovererSuite(t *testing.T) {
	suite.Run(t, new(IdentityNodeDiscovererSuite))
}

func (s *IdentityNodeDiscovererSuite) TestFindNodes() {
	discoverer := NewIdentityNodeDiscoverer(IdentityNodeDiscovererParams{
		Host: s.node1,
	})
	peerIDs, err := discoverer.FindNodes(context.Background(), model.Job{})
	s.NoError(err)

	peerIDStrings := make([]string, len(peerIDs))
	for i, p := range peerIDs {
		peerIDStrings[i] = p.PeerInfo.ID.String()
	}
	s.ElementsMatch([]string{s.node1.ID().String(), s.node2.ID().String()}, peerIDStrings)
}
