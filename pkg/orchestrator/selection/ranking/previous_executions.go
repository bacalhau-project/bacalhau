package ranking

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
// - Rank -1: Node has executed the job more than once, has rejected a bid, or produced a invalid result
func (s *PreviousExecutionsNodeRanker) RankNodes(ctx context.Context, job model.Job, nodes []model.NodeInfo) ([]orchestrator.NodeRank, error) {
	ranks := make([]orchestrator.NodeRank, len(nodes))
	previousExecutors := make(map[string]int)
	toFilterOut := make(map[string]bool)
	jobState, err := s.jobStore.GetJobState(ctx, job.ID())
	if err == nil {
		for _, execution := range jobState.Executions {
			if _, ok := previousExecutors[execution.NodeID]; !ok {
				previousExecutors[execution.NodeID] = 0
			}
			previousExecutors[execution.NodeID]++
			if !execution.State.IsDiscarded() {
				toFilterOut[execution.NodeID] = true
			}
			if execution.State == model.ExecutionStateAskForBidRejected {
				toFilterOut[execution.NodeID] = true
			}
		}
	}
	for i, node := range nodes {
		rank := 3 * orchestrator.RankPreferred
		reason := "job not executed yet"
		if previousExecutions, ok := previousExecutors[node.PeerInfo.ID.String()]; ok {
			if previousExecutions > 1 {
				rank = orchestrator.RankUnsuitable
				reason = "job already executed on this node more than once"
			} else if _, filterOut := toFilterOut[node.PeerInfo.ID.String()]; filterOut {
				rank = orchestrator.RankUnsuitable
				reason = "job rejected or produced invalid result"
			} else {
				rank = orchestrator.RankPossible
				reason = "job already executed on this node"
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
