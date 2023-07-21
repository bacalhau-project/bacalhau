package ranking

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type RandomNodeRankerParams struct {
	RandomnessRange int
}

// RandomNodeRanker assigns a random rank to each node to allow the orchestrator to select random top nodes
// for job execution.
type RandomNodeRanker struct {
	randomnessRange *big.Int
}

func NewRandomNodeRanker(params RandomNodeRankerParams) *RandomNodeRanker {
	if params.RandomnessRange <= orchestrator.RankPossible {
		panic(fmt.Sprintf("randomness range must be >= %d: %d", orchestrator.RankPossible, params.RandomnessRange))
	}
	return &RandomNodeRanker{
		randomnessRange: big.NewInt(int64(params.RandomnessRange)),
	}
}

func (s *RandomNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	for i, node := range nodes {
		rank, err := s.getRandomRank()
		if err != nil {
			return nil, err
		}
		ranks[i] = orchestrator.NodeRank{
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
