//go:build unit || !integration

package discovery

import (
	"context"
	"errors"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type ChainedSuite struct {
	suite.Suite
	chain   *Chain
	peerID1 models.NodeInfo
	peerID2 models.NodeInfo
	peerID3 models.NodeInfo
}

func (s *ChainedSuite) SetupSuite() {
	s.peerID1 = models.NodeInfo{NodeID: "peerID1"}
	s.peerID2 = models.NodeInfo{NodeID: "peerID2"}
	s.peerID3 = models.NodeInfo{NodeID: "peerID3"}
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
	s.ElementsMatch([]models.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

func (s *ChainedSuite) TestListNodes_Overlap() {
	s.chain.Add(NewFixedDiscoverer(s.peerID1, s.peerID2))
	s.chain.Add(NewFixedDiscoverer(s.peerID2, s.peerID3))

	peerIDs, err := s.chain.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
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
	s.ElementsMatch([]models.NodeInfo{s.peerID1, s.peerID2, s.peerID3}, peerIDs)
}

// node discoverer that always returns an error
type badDiscoverer struct{}

func newBadDiscoverer() *badDiscoverer {
	return &badDiscoverer{}
}

func (b *badDiscoverer) ListNodes(context.Context) ([]models.NodeInfo, error) {
	return nil, errors.New("bad discoverer")
}
