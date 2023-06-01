//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type ChainSuite struct {
	suite.Suite
	chain   *Chain
	peerID1 model.NodeInfo
	peerID2 model.NodeInfo
	peerID3 model.NodeInfo
}

func (s *ChainSuite) SetupSuite() {
	s.peerID1 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID1")}}
	s.peerID2 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID2")}}
	s.peerID3 = model.NodeInfo{PeerInfo: peer.AddrInfo{ID: peer.ID("peerID3")}}
}

func (s *ChainSuite) SetupTest() {
	s.chain = NewChain()
}

func TestChainSuite(t *testing.T) {
	suite.Run(t, new(ChainSuite))
}

func (s *ChainSuite) TestRankNodes() {
	s.chain.Add(NewFixedRanker(0, 10, 10))
	s.chain.Add(NewFixedRanker(0, 0, 0))
	s.chain.Add(NewFixedRanker(0, 10, 20))

	ranks, err := s.chain.RankNodes(context.Background(), model.Job{}, []model.NodeInfo{s.peerID1, s.peerID2, s.peerID3})
	s.NoError(err)
	s.Equal(3, len(ranks))
	assertEquals(s.T(), ranks, "peerID1", 0)
	assertEquals(s.T(), ranks, "peerID2", 20)
	assertEquals(s.T(), ranks, "peerID3", 30)
}

func (s *ChainSuite) TestRankNodes_Negative() {
	s.chain.Add(NewFixedRanker(10, 10, 100))
	s.chain.Add(NewFixedRanker(0, 0, -1))
	s.chain.Add(NewFixedRanker(0, 10, 1000))

	ranks, err := s.chain.RankNodes(context.Background(), model.Job{}, []model.NodeInfo{s.peerID1, s.peerID2, s.peerID3})
	s.NoError(err)
	s.Equal(3, len(ranks))
	assertEquals(s.T(), ranks, "peerID1", 10)
	assertEquals(s.T(), ranks, "peerID2", 20)
	assertEquals(s.T(), ranks, "peerID3", -1)
}

func (s *ChainSuite) TestRankNodes_AllNegative() {
	s.chain.Add(NewFixedRanker(-99, -99, -99))
	s.chain.Add(NewFixedRanker(-1, -1, -1))
	s.chain.Add(NewFixedRanker(-999, 999, 999))

	ranks, err := s.chain.RankNodes(context.Background(), model.Job{}, []model.NodeInfo{s.peerID1, s.peerID2, s.peerID3})
	s.NoError(err)
	s.Equal(3, len(ranks))
	assertEquals(s.T(), ranks, "peerID1", -1)
	assertEquals(s.T(), ranks, "peerID2", -1)
	assertEquals(s.T(), ranks, "peerID3", -1)
}
