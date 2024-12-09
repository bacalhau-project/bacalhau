package ranking

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type OverSubscriptionNodeRanker struct {
	factor float64
}

func NewOverSubscriptionNodeRanker(factor float64) (*OverSubscriptionNodeRanker, error) {
	err := validate.IsGreaterOrEqual(factor, 1,
		"over subscription factor %f must be greater or equal to 1", factor)
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
	jobResourceUsage, err := job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("failed to convert job resources config to resources: %w", err)
	}

	ranks := make([]orchestrator.NodeRank, len(nodes))
	for i, node := range nodes {
		var rank int
		var reason string

		if node.ComputeNodeInfo.MaxCapacity.IsZero() {
			rank = orchestrator.RankUnsuitable
			reason = "node queue usage is unknown"
		} else {
			// overSubscriptionCapacity is the capacity at which the node can accept more jobs
			overSubscriptionCapacity := node.ComputeNodeInfo.MaxCapacity.Multiply(s.factor)

			// totalUsage is the sub of actively running capacity, queued capacity and new job resources
			totalUsage := node.ComputeNodeInfo.MaxCapacity.
				Sub(node.ComputeNodeInfo.AvailableCapacity).
				Add(node.ComputeNodeInfo.QueueUsedCapacity).
				Add(*jobResourceUsage)

			if totalUsage.LessThanEq(*overSubscriptionCapacity) {
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
