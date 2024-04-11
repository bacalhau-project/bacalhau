//go:build unit || !integration

package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// TODO wtf are we even testing here that the chain discoverer respects errors!??
// remove the chain discoverer and the discoverer interface, using the nodeStore is far simpler.

type ChainedSuite struct {
	suite.Suite
	chain   *Chain
	peerID1 models.NodeState
	peerID2 models.NodeState
	peerID3 models.NodeState
}

const (
	Peer1 = "peerID1"
	Peer2 = "peerID2"
	Peer3 = "peerID3"
)

func (s *ChainedSuite) SetupSuite() {
	s.peerID1 = models.NodeState{
		Info: models.NodeInfo{NodeID: Peer1},
	}
	s.peerID1 = models.NodeState{
		Info: models.NodeInfo{NodeID: Peer2},
	}
	s.peerID1 = models.NodeState{
		Info: models.NodeInfo{NodeID: Peer3},
	}
}

func (s *ChainedSuite) SetupTest() {
	s.chain = NewChain(false) // don't ignore errors
}

func TestChainedSuite(t *testing.T) {
	suite.Run(t, new(ChainedSuite))
}

func (s *ChainedSuite) TestListNodes() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1))
	s.chain.Add(NewFixedDiscoverer(s.peerID2))
	s.chain.Add(NewFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestListNodes_Overlap() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(NewFixedDiscoverer(s.peerID2, s.peerID3))

	peerIDs, err := s.chain.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestHandle_Error() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(NewFixedDiscoverer(s.peerID3))
	_, err := s.chain.ListNodes(context.Background())
	s.Error(err)
}

func (s *ChainedSuite) TestHandle_IgnoreError() {
	s.chain.ignoreErrors = true
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(newBadDiscoverer())
	s.chain.Add(NewFixedDiscoverer(s.peerID3))

	peerIDs, err := s.chain.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

// node discoverer that always returns an error
type badDiscoverer struct{}

func newBadDiscoverer() *badDiscoverer {
	return &badDiscoverer{}
}

func (b *badDiscoverer) ListNodes(context.Context) ([]models.NodeState, error) {
	return nil, errors.New("bad discoverer")
}
