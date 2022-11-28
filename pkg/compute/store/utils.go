package store

import (
	"context"

	"github.com/rs/zerolog/log"
)

// GetActiveExecution returns the active execution for a given shard.
// In case of a bug where we have more than a single active execution, the latest one is returned
func GetActiveExecution(ctx context.Context, s ExecutionStore, shardID string) (Execution, error) {
	executions, err := s.GetExecutions(ctx, shardID)
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
			"Found %d active executions for shard %s. Selecting the latest one", activeExecutionsCount, shardID)
	}

	return activeExecution, nil
}

func ValidateNewExecution(ctx context.Context, execution Execution) error {
	if execution.State != ExecutionStateCreated {
		return NewErrInvalidExecutionState(execution.ID, execution.State, ExecutionStateCreated)
	}
	if execution.Version != 1 {
		return NewErrInvalidExecutionVersion(execution.ID, execution.Version, 1)
	}

	return nil
}
