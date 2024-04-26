package ranking

import (
	"context"
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/dustin/go-humanize"
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
				reason = s.formatReason(*jobResourceUsage, node.ComputeNodeInfo.MaxJobRequirements)
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

const perResourceReason = "more %s (%s) than the maximum available (%s)"

func (s *MaxUsageNodeRanker) formatReason(requested, maximum models.Resources) string {
	reasons := make([]string, 0, 4) //nolint:gomnd  // number of resources
	if requested.CPU > maximum.CPU {
		reasons = append(reasons, fmt.Sprintf(perResourceReason, "CPU",
			fmt.Sprint(requested.CPU),
			fmt.Sprint(maximum.CPU),
		))
	}
	if requested.Memory > maximum.Memory {
		reasons = append(reasons, fmt.Sprintf(perResourceReason, "memory",
			humanize.Bytes(requested.Memory),
			humanize.Bytes(maximum.Memory),
		))
	}
	if requested.Disk > maximum.Disk {
		reasons = append(reasons, fmt.Sprintf(perResourceReason, "disk",
			humanize.Bytes(requested.Disk),
			humanize.Bytes(maximum.Disk),
		))
	}
	if requested.GPU > maximum.GPU {
		reasons = append(reasons, fmt.Sprintf(perResourceReason, "GPUs",
			fmt.Sprint(requested.GPU),
			fmt.Sprint(maximum.GPU),
		))
	}
	return fmt.Sprintf("job requires %s", strings.Join(reasons, " and "))
}
