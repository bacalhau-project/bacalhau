package verifier

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func ValidateExecutions(job model.Job, executions []model.ExecutionState) error {
	// minimum number of executions that should be present
	minCount := system.Min(job.Spec.Deal.Confidence, job.Spec.Deal.Concurrency)
	if len(executions) < minCount {
		return NewErrInsufficientExecutions(job.ID(), minCount, len(executions))
	}

	// all executions should match the job
	// all executions should be in a valid state
	for _, execution := range executions {
		if execution.JobID != job.ID() {
			return NewErrMismatchingExecution(job.ID(), execution.ID())
		}
		if execution.State != model.ExecutionStateResultProposed {
			return NewErrInvalidExecutionState(execution.ID(), execution.State)
		}
	}

	return nil
}
