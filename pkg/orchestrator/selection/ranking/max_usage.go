package ranking

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

type MaxUsageNodeRanker struct {
}

func NewMaxUsageNodeRanker() *MaxUsageNodeRanker {
	return &MaxUsageNodeRanker{}
}

// RankNodes ranks nodes based on the MaxJobRequirements the compute nodes are accepting:
// - Rank 10: Node is accepting MaxJobRequirements that are equal or higher than the job requirements.
// - Rank -1: Node is accepting MaxJobRequirements that are lower than the job requirements.
// - Rank 0: Node MaxJobRequirements are not set, or the node was discovered not through nodeInfoPublisher (e.g. identity protocol)
func (s *MaxUsageNodeRanker) RankNodes(ctx context.Context, job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	jobResourceUsage, err := job.Task().ResourcesConfig.ToResources()
	if err != nil {
		return nil, fmt.Errorf("failed to convert job resources config to resources: %w", err)
	}
	jobResourceUsageSet := !jobResourceUsage.IsZero()
	for i, node := range nodes {
		rank := orchestrator.RankPossible
		reason := "max job resource requirements not set or unknown"
		if jobResourceUsageSet && node.ComputeNodeInfo != nil {
			if jobResourceUsage.LessThanEq(node.ComputeNodeInfo.MaxJobRequirements) {
				rank = orchestrator.RankPreferred
				reason = "job requires less resources than are available"
			} else {
				rank = orchestrator.RankUnsuitable
				reason = fmt.Sprintf(
					"job requires more resources %s than are available per job %s",
					jobResourceUsage.String(),
					node.ComputeNodeInfo.MaxJobRequirements.String(),
				)
			}
		}
		ranks[i] = orchestrator.NodeRank{
			NodeInfo: node,
			Rank:     rank,
			Reason:   reason,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}
	return ranks, nil
}
