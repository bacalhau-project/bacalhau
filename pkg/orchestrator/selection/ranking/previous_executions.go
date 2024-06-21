package ranking

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type PreviousExecutionsNodeRankerParams struct {
	JobStore jobstore.Store
}
type PreviousExecutionsNodeRanker struct {
	jobStore jobstore.Store
}

func NewPreviousExecutionsNodeRanker(params PreviousExecutionsNodeRankerParams) *PreviousExecutionsNodeRanker {
	return &PreviousExecutionsNodeRanker{
		jobStore: params.JobStore,
	}
}

// RankNodes ranks nodes based on whether the node has already executed the job, which is useful when ranking
// nodes when handling retries:
// - Rank 30: Node has never executed the job.
// - Rank 30-n: Node has already executed the job n times.
// - Rank -1: Node has rejected a bid
func (s *PreviousExecutionsNodeRanker) RankNodes(ctx context.Context,
	job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	previousExecutors := make(map[string]int)
	toFilterOut := make(map[string]bool)
	executions, err := s.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
	if err == nil {
		for _, execution := range executions {
			if _, ok := previousExecutors[execution.NodeID]; !ok {
				previousExecutors[execution.NodeID] = 0
			}
			previousExecutors[execution.NodeID]++
			if execution.ComputeState.StateType == models.ExecutionStateAskForBidRejected {
				toFilterOut[execution.NodeID] = true
			}
		}
	}
	for i, node := range nodes {
		rank := 3 * orchestrator.RankPreferred
		reason := "job not executed yet"
		if previousExecutions, ok := previousExecutors[node.ID()]; ok {
			if previousExecutions > 1 {
				rank = math.Max(orchestrator.RankPossible, rank-previousExecutions)
				reason = fmt.Sprintf("job already executed on this node %d times", previousExecutions)
			} else if _, filterOut := toFilterOut[node.ID()]; filterOut {
				rank = orchestrator.RankUnsuitable
				reason = "job rejected"
			}
		}
		ranks[i] = orchestrator.NodeRank{
			NodeInfo: node,
			Rank:     rank,
			Reason:   reason,
			// No logic in here (yet) that will ignore failed executions after e.g. a period of time.
			// If we introduce that, this should become true.
			Retryable: false,
		}
		log.Ctx(ctx).Trace().Object("Rank", ranks[i]).Msg("Ranked node")
	}
	return ranks, nil
}
