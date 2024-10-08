//go:generate mockgen --source types.go --destination mocks.go --package store
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type LocalExecutionState struct {
	Execution       *models.Execution
	PublishedResult *models.SpecConfig
	RunOutput       *models.RunCommandResult
	RequesterNodeID string
	State           LocalExecutionStateType
	Revision        int
	CreateTime      time.Time
	UpdateTime      time.Time
	LatestComment   string
}

func NewLocalExecutionState(execution *models.Execution, requesterNodeID string) *LocalExecutionState {
	now := time.Now().UTC()
	return &LocalExecutionState{
		Execution:       execution,
		RequesterNodeID: requesterNodeID,
		State:           ExecutionStateCreated,
		Revision:        1,
		CreateTime:      now,
		UpdateTime:      now,
	}
}

// Normalize normalizes the execution state
func (e *LocalExecutionState) Normalize() {
	if e.Execution == nil {
		return
	}
	e.Execution.Normalize()
}

// string returns a string representation of the execution
func (e *LocalExecutionState) String() string {
	return fmt.Sprintf("{ID: %s, Job: %s}", e.Execution.ID, e.Execution.JobID)
}

// ToSummary returns a summary of the execution
func (e *LocalExecutionState) ToSummary() ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:        e.Execution.ID,
		JobID:              e.Execution.JobID,
		State:              e.State.String(),
		AllocatedResources: *e.Execution.AllocatedResources,
	}
}

type LocalStateHistory struct {
	ExecutionID   string
	PreviousState LocalExecutionStateType
	NewState      LocalExecutionStateType
	NewRevision   int
	Comment       string
	Time          time.Time
}

// Summary of an execution that is used in logging and debugging.
type ExecutionSummary struct {
	ExecutionID        string
	JobID              string
	State              string
	AllocatedResources models.AllocatedResources
}

type UpdateExecutionStateRequest struct {
	ExecutionID      string
	NewState         LocalExecutionStateType
	ExpectedStates   []LocalExecutionStateType
	ExpectedRevision int
	PublishedResult  *models.SpecConfig
	RunOutput        *models.RunCommandResult
	Comment          string
}

// Validate checks if the condition matches the given execution
func (condition UpdateExecutionStateRequest) Validate(localExecutionState LocalExecutionState) error {
	execution := localExecutionState.Execution
	if len(condition.ExpectedStates) > 0 {
		validState := false
		for _, s := range condition.ExpectedStates {
			if s == localExecutionState.State {
				validState = true
				break
			}
		}
		if !validState {
			return NewErrInvalidExecutionState(execution.ID, localExecutionState.State, condition.ExpectedStates...)
		}
	}

	if condition.ExpectedRevision != 0 && condition.ExpectedRevision != localExecutionState.Revision {
		return NewErrInvalidExecutionRevision(execution.ID, localExecutionState.Revision, condition.ExpectedRevision)
	}
	return nil
}

// ExecutionStore A metadata store of job executions handled by the current compute node
type ExecutionStore interface {
	// GetExecution returns the execution for a given id
	GetExecution(ctx context.Context, id string) (LocalExecutionState, error)
	// GetExecutions returns all the executions for a given job
	GetExecutions(ctx context.Context, jobID string) ([]LocalExecutionState, error)
	// GetLiveExecutions gets an array of the executions currently in the
	// active state (ExecutionStateBidAccepted)
	GetLiveExecutions(ctx context.Context) ([]LocalExecutionState, error)
	// GetExecutionHistory returns the history of an execution
	GetExecutionHistory(ctx context.Context, id string) ([]LocalStateHistory, error)
	// CreateExecution creates a new execution for a given job
	CreateExecution(ctx context.Context, execution LocalExecutionState) error
	// UpdateExecutionState updates the execution state
	UpdateExecutionState(ctx context.Context, request UpdateExecutionStateRequest) error
	// DeleteExecution deletes an execution
	DeleteExecution(ctx context.Context, id string) error
	// GetExecutionCount returns a count of all executions that are in the specified
	// state
	GetExecutionCount(ctx context.Context, state LocalExecutionStateType) (uint64, error)
	// GetEventStore returns the event store for the execution store
	GetEventStore() watcher.EventStore
	// Close provides the opportunity for the underlying store to cleanup
	// any resources as the compute node is shutting down
	Close(ctx context.Context) error
}
