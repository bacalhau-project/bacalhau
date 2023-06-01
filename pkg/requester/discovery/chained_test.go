//go:build unit || !integration

package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type ChainedSuite struct {
	suite.Suite
	chain   *Chain
	peerID1 model.NodeInfo
	peerID2 model.NodeInfo
	peerID3 model.NodeInfo
}

func (s *ChainedSuite) SetupSuite() {
	s.peerID1 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID1")}}
	s.peerID2 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID2")}}
	s.peerID3 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID3")}}
}

func (s *ChainedSuite) SetupTest() {
	s.chain = NewChain(false) // don't ignore errors
}

func TestChainedSuite(t *testing.T) {
	suite.Run(t, new(ChainedSuite))
}

func (s *ChainedSuite) TestFindNodes() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1))
	s.chain.Add(NewFixedDiscoverer(s.peerID2))
	s.chain.Add(NewFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]model.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestFindNodes_Overlap() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(NewFixedDiscoverer(s.peerID2, s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]model.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestHandle_Error() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(NewFixedDiscoverer(s.peerID3))
	_, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.Error(err)
}

func (s *ChainedSuite) TestHandle_IgnoreError() {
	s.chain.ignoreErrors = true
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(NewFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]model.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

// node discoverer that always returns an error
type badDiscoverer struct{}

func newBadDiscoverer() *badDiscoverer {
	return &badDiscoverer{}
}

func (b *badDiscoverer) FindNodes(context.Context, model.Job) ([]model.NodeInfo, error) {
	return nil, errors.New("bad discoverer")
}

func (b *badDiscoverer) ListNodes(context.Context) ([]model.NodeInfo, error) {
	return nil, errors.New("bad discoverer")
}
