package ranking

import (
	"context"
	"crypto/rand"
	"math/big"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
)

type RandomNodeRankerParams struct {
	RandomnessRange int
}

// RandomNodeRanker assigns a random rank to each node to allow the requester to select random top nodes
// for job execution.
type RandomNodeRanker struct {
	randomnessRange *big.Int
}

func NewRandomNodeRanker(params RandomNodeRankerParams) *RandomNodeRanker {
	return &RandomNodeRanker{
		randomnessRange: big.NewInt(int64(params.RandomnessRange)),
	}
}

func (s *RandomNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []peer.ID) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	for i, node := range nodes {
		rank, err := s.getRandomRank()
		if err != nil {
			return nil, err
		}
		ranks[i] = requester.NodeRank{
			ID:   node,
			Rank: rank,
		}
	}
	return ranks, nil
}

func (s *RandomNodeRanker) getRandomRank() (int, error) {
	nBig, err := rand.Int(rand.Reader, s.randomnessRange)
	if err != nil {
		return 0, err
	}
	return int(nBig.Int64()), nil
}
