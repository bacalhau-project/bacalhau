package verifier

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func ValidateExecutions(request VerifierRequest) error {
	// minimum number of executions that should be present
	minCount := system.Min(request.Deal.GetConfidence(), request.Deal.GetConcurrency())
	if len(request.Executions) < minCount {
		return NewErrInsufficientExecutions(request.JobID, minCount, len(request.Executions))
	}

	// all executions should match the job
	// all executions should be in a valid state
	for _, execution := range request.Executions {
		if execution.JobID != request.JobID {
			return NewErrMismatchingExecution(request.JobID, execution.ID())
		}
		if execution.State != model.ExecutionStateResultProposed {
			return NewErrInvalidExecutionState(execution.ID(), execution.State)
		}
	}

	return nil
}
