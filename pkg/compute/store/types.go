//go:generate mockgen --source types.go --destination mocks.go --package store
package store

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/lib/boltdblib"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// ExecutionStore A metadata store of job executions handled by the current compute node
type ExecutionStore interface {
	// BeginTx starts a new transaction and returns a transactional context
	BeginTx(ctx context.Context) (boltdblib.TxContext, error)
	// GetExecution returns the execution for a given id
	GetExecution(ctx context.Context, id string) (*models.Execution, error)
	// GetExecutions returns all the executions for a given job
	GetExecutions(ctx context.Context, jobID string) ([]*models.Execution, error)
	// GetLiveExecutions gets an array of the executions currently in the
	// active state (ExecutionStateBidAccepted)
	GetLiveExecutions(ctx context.Context) ([]*models.Execution, error)
	// AddExecutionEvent adds an event to the execution
	AddExecutionEvent(ctx context.Context, executionID string, events ...*models.Event) error
	// GetExecutionEvents returns the history of an execution
	GetExecutionEvents(ctx context.Context, executionID string) ([]*models.Event, error)
	// CreateExecution creates a new execution for a given job
	CreateExecution(ctx context.Context, execution models.Execution, events ...*models.Event) error
	// UpdateExecutionState updates the execution state
	UpdateExecutionState(ctx context.Context, request UpdateExecutionRequest) error
	// DeleteExecution deletes an execution
	DeleteExecution(ctx context.Context, id string) error
	// GetExecutionCount returns a count of all executions that are in the specified state
	GetExecutionCount(ctx context.Context, state models.ExecutionStateType) (uint64, error)
	// GetEventStore returns the event store for the execution store
	GetEventStore() watcher.EventStore
	// Checkpoint saves the last sequence number processed
	Checkpoint(ctx context.Context, name string, sequenceNumber uint64) error
	// GetCheckpoint returns the last sequence number processed
	GetCheckpoint(ctx context.Context, name string) (uint64, error)
	// Close provides the opportunity for the underlying store to cleanup
	// any resources as the compute node is shutting down
	Close(ctx context.Context) error
}

type UpdateExecutionRequest struct {
	ExecutionID string
	Condition   UpdateExecutionCondition
	NewValues   models.Execution
	Events      []*models.Event
}

type UpdateExecutionCondition struct {
	ExpectedStates   []models.ExecutionStateType
	ExpectedRevision uint64
	UnexpectedStates []models.ExecutionStateType
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionCondition) Validate(execution *models.Execution) error {
	if len(condition.ExpectedStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedStates {
			if s == execution.ComputeState.StateType {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType, condition.ExpectedStates...)
		}
	}

	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != execution.Revision {
		return NewErrInvalidExecutionRevision(execution.ID, execution.Revision, condition.ExpectedRevision)
	}
	if len(condition.UnexpectedStates) > 0 {
		for _, s := range condition.UnexpectedStates {
			if s == execution.ComputeState.StateType {
				return NewErrInvalidExecutionState(execution.ID, execution.ComputeState.StateType)
			}
		}
	}
	return nil
}
