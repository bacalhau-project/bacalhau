package ranking

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"
)

type LabelsNodeRanker struct {
}

func NewLabelsNodeRanker() *LabelsNodeRanker {
	return &LabelsNodeRanker{}
}

// RankNodes ranks nodes based on the node labels and job selectors:
// - Rank 1: Selectors match node labels.
// - Rank -1: Selectors don't match node labels.
// - Rank 0: Job selectors are not set.
func (s *LabelsNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	var selector labels.Selector
	if len(job.Spec.NodeSelectors) > 0 {
		requirements, err := model.FromLabelSelectorRequirements(job.Spec.NodeSelectors...)
		if err != nil {
			return nil, err
		}
		selector = labels.NewSelector().Add(requirements...)
	}
	for i, node := range nodes {
		rank := 0
		if selector != nil {
			if selector.Matches(labels.Set(node.Labels)) {
				rank = 1
			} else {
				log.Ctx(ctx).Trace().Msgf("filtering node %s with labels %s doesn't match selectors %+v",
					node.PeerInfo.ID, node.Labels, job.Spec.NodeSelectors)
				rank = -1
			}
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
		}
	}
	return ranks, nil
}
