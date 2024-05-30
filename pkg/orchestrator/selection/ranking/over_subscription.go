package ranking

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type OverSubscriptionNodeRanker struct {
	factor float64
}

func NewOverSubscriptionNodeRanker(factor float64) (*OverSubscriptionNodeRanker, error) {
	err := validate.IsGreaterOrEqualToZero(factor,
		"over subscription factor %f must be greater or equal to zero", factor)
	if err != nil {
		return nil, err
	}
	return &OverSubscriptionNodeRanker{factor: factor}, nil
}

// RankNodes ranks nodes based on the ratio of queued capacity to total capacity.
// - Rank -1: If the ratio of is greater than the factor, the node is considered over-subscribed.
// - Rank 0: If the node is not over-subscribed.
func (s *OverSubscriptionNodeRanker) RankNodes(
	ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	for i, node := range nodes {
		var rank int
		var reason string

		if node.ComputeNodeInfo == nil {
			rank = orchestrator.RankUnsuitable
			reason = "node queue usage is unknown"
		} else {
			queueCapacity := node.ComputeNodeInfo.MaxCapacity.Multiply(s.factor)
			if node.ComputeNodeInfo.QueueUsedCapacity.LessThanEq(*queueCapacity) {
				rank = orchestrator.RankPossible
				reason = "node is not over-subscribed"
			} else {
				rank = orchestrator.RankUnsuitable
				reason = "node busy with available capacity " + node.ComputeNodeInfo.AvailableCapacity.String()
				if !node.ComputeNodeInfo.QueueUsedCapacity.IsZero() {
					reason += " and queue capacity " + node.ComputeNodeInfo.QueueUsedCapacity.String()
				}
			}
		}

		ranks[i] = orchestrator.NodeRank{
			NodeInfo:  node,
			Rank:      rank,
			Reason:    reason,
			Retryable: true,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}
	return ranks, nil
}
