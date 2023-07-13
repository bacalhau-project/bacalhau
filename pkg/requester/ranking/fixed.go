package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
)

// node Ranker that always returns the same set of nodes
type fixedRanker struct {
	ranks []int
}

func NewFixedRanker(ranks ...int) *fixedRanker {
	return &fixedRanker{
		ranks: ranks,
	}
}

func (f *fixedRanker) RankNodes(_ context.Context, _ model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	for i, rank := range f.ranks {
		ranks[i] = requester.NodeRank{
			NodeInfo: nodes[i],
			Rank:     rank,
		}
	}
	return ranks, nil
}
