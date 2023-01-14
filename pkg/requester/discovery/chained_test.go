package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type ChainedSuite struct {
	suite.Suite
	chain   *Chained
	peerID1 peer.ID
	peerID2 peer.ID
	peerID3 peer.ID
}

func (s *ChainedSuite) SetupSuite() {
	s.peerID1 = peer.ID("peerID1")
	s.peerID2 = peer.ID("peerID2")
	s.peerID3 = peer.ID("peerID3")
}

func (s *ChainedSuite) SetupTest() {
	s.chain = NewChained(false) // don't ignore errors
}

func TestChainedSuite(t *testing.T) {
	suite.Run(t, new(ChainedSuite))
}

func (s *ChainedSuite) TestFindNodes() {
	s.chain.Add(newFixedDiscoverer(s.peerID1))
	s.chain.Add(newFixedDiscoverer(s.peerID2))
	s.chain.Add(newFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]peer.ID{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestFindNodes_Overlap() {
	s.chain.Add(newFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newFixedDiscoverer(s.peerID2, s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]peer.ID{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestHandle_Error() {
	s.chain.Add(newFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(newFixedDiscoverer(s.peerID3))
	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.Error(err)
	s.Empty(peerIDs)
}

func (s *ChainedSuite) TestHandle_IgnoreError() {
	s.chain.ignoreErrors = true
	s.chain.Add(newFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(newFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.ElementsMatch([]peer.ID{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

// node discoverer that always returns the same set of nodes
type fixedDiscoverer struct {
	peerIDs []peer.ID
}

func newFixedDiscoverer(peerIDs ...peer.ID) *fixedDiscoverer {
	return &fixedDiscoverer{
		peerIDs: peerIDs,
	}
}

func (f *fixedDiscoverer) FindNodes(ctx context.Context, job model.Job) ([]peer.ID, error) {
	return f.peerIDs, nil
}

// node discoverer that always returns an error
type badDiscoverer struct{}

func newBadDiscoverer() *badDiscoverer {
	return &badDiscoverer{}
}

func (b *badDiscoverer) FindNodes(ctx context.Context, job model.Job) ([]peer.ID, error) {
	return nil, errors.New("bad discoverer")
}
