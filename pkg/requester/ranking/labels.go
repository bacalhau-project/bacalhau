package ranking

import (
	"context"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/labels"
)

// FavourNodeSelectorPrefix labels prefixed with this string will be considered as a "favour" selector that prioritizes those nodes,
// instead of a "must" selector that filters out nodes that don't match.
const FavourNodeSelectorPrefix = "favour_"

type LabelsNodeRanker struct {
}

func NewLabelsNodeRanker() *LabelsNodeRanker {
	return &LabelsNodeRanker{}
}

// RankNodes ranks nodes based on the node labels and job selectors:
// - Rank 20: Selectors with `favour_` prefix and matching node labels
// - Rank 10: Selectors match node labels.
// - Rank -1: Selectors don't match node labels.
// - Rank 0: Job selectors are not set.
func (s *LabelsNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	mustSelector := labels.NewSelector()
	favourSelector := labels.NewSelector()
	if len(job.Spec.NodeSelectors) > 0 {
		requirements, err := model.FromLabelSelectorRequirements(job.Spec.NodeSelectors...)
		if err != nil {
			return nil, err
		}
		for _, requirement := range requirements {
			if strings.HasPrefix(requirement.Key(), FavourNodeSelectorPrefix) {
				trimmedRequirement, err2 := labels.NewRequirement(
					strings.TrimPrefix(requirement.Key(), FavourNodeSelectorPrefix),
					requirement.Operator(),
					requirement.Values().List())
				if err2 != nil {
					return nil, err2
				}
				favourSelector = favourSelector.Add(*trimmedRequirement)
			} else {
				mustSelector = mustSelector.Add(requirement)
			}
		}
	}
	for i, node := range nodes {
		rank := 0
		if !mustSelector.Empty() {
			if mustSelector.Matches(labels.Set(node.Labels)) {
				rank = 10
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

	if !favourSelector.Empty() {
		for i, rank := range ranks {
			if rank.MeetsRequirement() && favourSelector.Matches(labels.Set(rank.NodeInfo.Labels)) {
				ranks[i].Rank += requester.RankPreferred
			}
		}
	}
	return ranks, nil
}
