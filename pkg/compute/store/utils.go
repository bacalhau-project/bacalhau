package store

import (
	"context"

	"github.com/rs/zerolog/log"
)

// GetActiveExecution returns the active execution for a given job.
// In case of a bug where we have more than a single active execution, the latest one is returned
func GetActiveExecution(ctx context.Context, s ExecutionStore, jobID string) (Execution, error) {
	executions, err := s.GetExecutions(ctx, jobID)
	if err != nil {
		return Execution{}, err
	}

	var activeExecution Execution
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

func ValidateNewExecution(execution Execution) error {
	if execution.State != ExecutionStateCreated {
		return NewErrInvalidExecutionState(execution.ID, execution.State, ExecutionStateCreated)
	}
	if execution.Version != 1 {
		return NewErrInvalidExecutionVersion(execution.ID, execution.Version, 1)
	}

	return nil
}
