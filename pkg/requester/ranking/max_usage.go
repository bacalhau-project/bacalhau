package ranking

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
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
func (s *MaxUsageNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]requester.NodeRank, error) {
	ranks := make([]requester.NodeRank, len(nodes))
	jobResourceUsage := capacity.ParseResourceUsageConfig(job.Spec.Resources)
	jobResourceUsageSet := !jobResourceUsage.IsZero()
	for i, node := range nodes {
		rank := requester.RankPossible
		reason := "max job resource requirements not set or unknown"
		if jobResourceUsageSet && node.ComputeNodeInfo != nil {
			if jobResourceUsage.LessThanEq(node.ComputeNodeInfo.MaxJobRequirements) {
				rank = requester.RankPreferred
				reason = "job requires less resources than are available"
			} else {
				rank = requester.RankUnsuitable
				reason = fmt.Sprintf(
					"job requires more resources %s than are available per job %s",
					jobResourceUsage,
					node.ComputeNodeInfo.MaxJobRequirements,
				)
			}
		}
		ranks[i] = requester.NodeRank{
			NodeInfo: node,
			Rank:     rank,
			Reason:   reason,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}
	return ranks, nil
}
