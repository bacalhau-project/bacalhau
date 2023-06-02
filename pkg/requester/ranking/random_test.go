//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type RandomNodeRankerSuite struct {
	suite.Suite
	RandomNodeRanker *RandomNodeRanker
}

func TestRandomNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(RandomNodeRankerSuite))
}

func (s *RandomNodeRankerSuite) TestRankNodes() {
	nodeCount := 100
	randomnessRange := 10

	var nodes []model.NodeInfo
	for i := 0; i < nodeCount; i++ {
		nodes = append(nodes, model.NodeInfo{
			PeerInfo: peer.AddrInfo{ID: peer.ID(rune(i))},
		})
	}
	s.RandomNodeRanker = NewRandomNodeRanker(RandomNodeRankerParams{RandomnessRange: randomnessRange})

	ranks, err := s.RandomNodeRanker.RankNodes(context.Background(), model.Job{}, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))

	uniqueRanks := make(map[int]struct{})
	for _, rank := range ranks {
		s.True(rank.Rank >= requester.RankPossible)
		s.True(rank.Rank <= randomnessRange)
		uniqueRanks[rank.Rank] = struct{}{}
	}

	s.True(len(uniqueRanks) > 1)
}

func (s *RandomNodeRankerSuite) TestRankNodes_NoRandomness() {
	defer func() {
		if r := recover(); r == nil {
			s.Fail("expected panic when randomness range is 0")
		}
	}()
	NewRandomNodeRanker(RandomNodeRankerParams{RandomnessRange: 0})
}

func (s *RandomNodeRankerSuite) TestRankNodes_NegativeRandomness() {
	defer func() {
		if r := recover(); r == nil {
			s.Fail("expected panic when randomness range is negative")
		}
	}()
	NewRandomNodeRanker(RandomNodeRankerParams{RandomnessRange: -1})
}
