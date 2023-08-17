package store

import (
	"context"

	"github.com/rs/zerolog/log"
)

// GetActiveExecution returns the active execution for a given job.
// In case of a bug where we have more than a single active execution, the latest one is returned
func GetActiveExecution(ctx context.Context, s ExecutionStore, jobID string) (LocalState, error) {
	executions, err := s.GetExecutions(ctx, jobID)
	if err != nil {
		return LocalState{}, err
	}

	var activeExecution LocalState
	var activeExecutionsCount int
	for _, execution := range executions {
		if execution.State.IsActive() {
			activeExecutionsCount++
			if activeExecutionsCount != 1 || execution.UpdateTime.After(activeExecution.UpdateTime) {
				activeExecution = execution
			}
		}
	}

	if activeExecutionsCount > 1 {
		log.Ctx(ctx).Warn().Msgf(
			"Found %d active executions for job %s. Selecting the latest one", activeExecutionsCount, jobID)
	}

	return activeExecution, nil
}

func ValidateNewExecution(localExecutionState LocalState) error {
	// state must be either created, or bid accepted if the execution is pre-approved
	if localExecutionState.State != ExecutionStateCreated && localExecutionState.State != ExecutionStateBidAccepted {
		return NewErrInvalidExecutionState(
			localExecutionState.Execution.ID, localExecutionState.State, ExecutionStateCreated, ExecutionStateBidAccepted)
	}
	if localExecutionState.Version != 1 {
		return NewErrInvalidExecutionVersion(localExecutionState.Execution.ID, localExecutionState.Version, 1)
	}

	return nil
}
