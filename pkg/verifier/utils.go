package verifier

import (
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

func ValidateExecutions(shard model.JobShard, executions []model.ExecutionState) error {
	// minimum number of executions that should be present
	minCount := system.Min(shard.Job.Spec.Deal.Confidence, shard.Job.Spec.Deal.Concurrency)
	if len(executions) < minCount {
		return NewErrInsufficientExecutions(shard.ID(), minCount, len(executions))
	}

	// all executions should match the shard
	// all executions should be in a valid state
	for _, execution := range executions {
		if execution.ShardID() != shard.ShardID() {
			return NewErrMismatchingExecution(shard.ShardID(), execution.ID())
		}
		if execution.State != model.ExecutionStateResultProposed {
			return NewErrInvalidExecutionState(execution.ID(), execution.State)
		}
	}

	return nil
}
