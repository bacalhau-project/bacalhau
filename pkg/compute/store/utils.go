package store

import (
	"context"
	"fmt"
	"sort"

	"github.com/rs/zerolog/log"
	"github.com/samber/lo"
)

// GetActiveExecution returns the active execution for a given job.
// In case of a bug where we have more than a single active execution, the latest one is returned
// where we expect that to be the most recent (that is GetExecutions returns the executions in
// time order).
func GetActiveExecution(ctx context.Context, s ExecutionStore, jobID string) (Execution, error) {
	executions, err := s.GetExecutions(ctx, jobID)
	if err != nil {
		return Execution{}, err
	}

	activeExecutions := lo.Filter(executions, func(item Execution, index int) bool {
		return item.State.IsActive()
	})
	activeExecutionsCount := len(activeExecutions)

	if activeExecutionsCount == 0 {
		return Execution{}, fmt.Errorf("no active executions found for job : %s", jobID)
	}

	if activeExecutionsCount == 1 {
		return activeExecutions[0], nil
	}

	log.Ctx(ctx).Warn().Msgf(
		"Found %d active executions for job %s. Selecting the latest one", activeExecutionsCount, jobID)

	// Ensure the activeExecutions are sorted by UpdateTime with earliest first
	sort.Slice(activeExecutions, func(i, j int) bool {
		return activeExecutions[i].UpdateTime.Before(activeExecutions[j].UpdateTime)
	})

	return lo.Last(activeExecutions)
}

func ValidateNewExecution(_ context.Context, execution Execution) error {
	if execution.State != ExecutionStateCreated {
		return NewErrInvalidExecutionState(execution.ID, execution.State, ExecutionStateCreated)
	}
	if execution.Version != 1 {
		return NewErrInvalidExecutionVersion(execution.ID, execution.Version, 1)
	}

	return nil
}
