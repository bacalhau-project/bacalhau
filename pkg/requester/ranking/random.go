package ranking

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
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
	if params.RandomnessRange <= requester.RankPossible {
		panic(fmt.Sprintf("randomness range must be >= %d: %d", requester.RankPossible, params.RandomnessRange))
	}
	return &RandomNodeRanker{
		randomnessRange: big.NewInt(int64(params.RandomnessRange)),
	}
}

func (s *RandomNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	for i, node := range nodes {
		rank, err := s.getRandomRank()
		if err != nil {
			return nil, err
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
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
