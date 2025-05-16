package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
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
// - Rank 0: Node has already executed the job.
// - Rank -1: Node has executed the job more than once or has rejected a bid
func (s *PreviousExecutionsNodeRanker) RankNodes(ctx context.Context,
	job models.Job, nodes []models.NodeInfo) ([]orchestrator.NodeRank, error) {
	// If it is a job update, what should I do ?
	//jobIsARerun := false
	currentRuntimeID := ""

	//if passedIsARerun, ok := ctx.Value("node-ranking/is-job-rerun").(bool); ok {
	//	jobIsARerun = passedIsARerun
	//}

	//node-ranking/current-runtime-id
	if passedIsRuntimeID, ok := ctx.Value("node-ranking/current-runtime-id").(string); ok {
		currentRuntimeID = passedIsRuntimeID
	}

	ranks := make([]orchestrator.NodeRank, len(nodes))
	previousExecutors := make(map[string]int)
	toFilterOut := make(map[string]bool)
	executions, err := s.jobStore.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})

	filteredExecutions := []models.Execution{}
	if currentRuntimeID != "" {
		for _, execution := range executions {
			if execution.RuntimeID == currentRuntimeID {
				filteredExecutions = append(filteredExecutions, execution)
			}
		}
	}
	executions = filteredExecutions

	// TODO:
	// Only care about the executions of this runtime ID
	// the runtime ID will be provided in the context like the
	// node-ranking/is-job-rerun about
	if err == nil {
		for _, execution := range executions {
			if _, ok := previousExecutors[execution.NodeID]; !ok {
				previousExecutors[execution.NodeID] = 0
			}
			previousExecutors[execution.NodeID]++
			//if !execution.IsDiscarded() {
			//	toFilterOut[execution.NodeID] = true
			//}
			if execution.ComputeState.StateType == models.ExecutionStateAskForBidRejected {
				toFilterOut[execution.NodeID] = true
			}
		}
	}
	for i, node := range nodes {
		rank := 3 * orchestrator.RankPreferred
		reason := "job not executed yet"
		if previousExecutions, ok := previousExecutors[node.ID()]; ok {
			//if jobIsARerun {
			//	reason = "job iteration is a rerun - treat it like a new run"
			//} else
			if previousExecutions > 1 {
				// this is the culprit
				rank = orchestrator.RankUnsuitable
				reason = "job already executed on this node more than once"
			} else if _, filterOut := toFilterOut[node.ID()]; filterOut {
				rank = orchestrator.RankUnsuitable
				reason = "job rejected"
			} else {
				rank = orchestrator.RankPossible
				reason = "job already executed on this node"
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
