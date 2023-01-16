package requester

import (
	"github.com/filecoin-project/bacalhau/pkg/model"
)

// A job that is holding compute capacity, which can be in bidding or running state.
type ScheduledJob struct {
	ShardID             string `json:"shardId"`
	State               string `json:"state"`
	BiddingNodesCount   int    `json:"biddingNodesCount"`
	CompletedNodesCount int    `json:"completedNodesCount"`
}

type ScheduledJobsInfoProviderParams struct {
	Name      string
	Scheduler *Scheduler
}

// ScheduledJobsInfoProvider provides DebugInfo about the currently running executions.
// The info can be used for logging, metric, or to handle /debug API implementation.
type ScheduledJobsInfoProvider struct {
	name      string
	scheduler *Scheduler
}

func NewScheduledJobsInfoProvider(params ScheduledJobsInfoProviderParams) *ScheduledJobsInfoProvider {
	return &ScheduledJobsInfoProvider{
		name:      params.Name,
		scheduler: params.Scheduler,
	}
}

func (r ScheduledJobsInfoProvider) GetDebugInfo() (model.DebugInfo, error) {
	scheduledJobs := make([]ScheduledJob, 0)

	for _, shardState := range r.scheduler.shardStateManager.shardStates {
		if shardState.currentState != shardCompleted && shardState.currentState != shardError {
			scheduledJobs = append(scheduledJobs, ScheduledJob{
				ShardID:             shardState.shard.ID(),
				State:               shardState.currentState.String(),
				BiddingNodesCount:   len(shardState.biddingNodes),
				CompletedNodesCount: len(shardState.completedNodes),
			})
		}
	}

	return model.DebugInfo{
		Component: r.name,
		Info:      scheduledJobs,
	}, nil
}

// compile-time check that we implement the interface
var _ model.DebugInfoProvider = (*ScheduledJobsInfoProvider)(nil)
