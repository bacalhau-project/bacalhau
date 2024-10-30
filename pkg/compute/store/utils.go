package store

import (
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

func ValidateNewExecution(execution *models.Execution) error {
	// state must be either created, or bid accepted if the execution is pre-approved
	if execution.ComputeState.StateType != models.ExecutionStateNew &&
		execution.ComputeState.StateType != models.ExecutionStateBidAccepted {
		return NewErrInvalidExecutionState(
			execution.ID, execution.ComputeState.StateType, models.ExecutionStateNew, models.ExecutionStateBidAccepted)
	}
	if execution.Revision > 1 {
		return NewErrInvalidExecutionRevision(execution.ID, execution.Revision, 1)
	}
	err := execution.Validate()
	if err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}
	if execution.Job == nil {
		return fmt.Errorf("CreateExecution failure: job is nil")
	}

	return nil
}
