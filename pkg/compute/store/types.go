//go:generate mockgen --source types.go --destination mocks.go --package store
package store

import (
	"context"
	"fmt"
	"time"
)

type LocalState struct {
	Execution     *models.Execution
	State         LocalStateType
	Version       int
	CreateTime    time.Time
	UpdateTime    time.Time
	LatestComment string
}

func NewLocalState(execution *models.Execution) *LocalState {
	return &LocalState{
		Execution:  execution,
		State:      ExecutionStateCreated,
		Version:    1,
		CreateTime: time.Now().UTC(),
		UpdateTime: time.Now().UTC(),
	}
}

// string returns a string representation of the execution
func (e *LocalState) String() string {
	return fmt.Sprintf("{ID: %s, Job: %s}", e.Execution.ID, e.Execution.Job.ID)
}

// ToSummary returns a summary of the execution
func (e *LocalState) ToSummary() ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:        e.Execution.ID,
		JobID:              e.Execution.Job.ID,
		State:              e.State.String(),
		AllocatedResources: e.Execution.AllocatedResources,
	}
}

type LocalStateHistory struct {
	ExecutionID   string
	PreviousState LocalStateType
	NewState      LocalStateType
	NewVersion    int
	Comment       string
	Time          time.Time
}

// Summary of an execution that is used in logging and debugging.
type ExecutionSummary struct {
	ExecutionID        string
	JobID              string
	State              string
	AllocatedResources *models.AllocatedResources
}

type UpdateExecutionStateRequest struct {
	ExecutionID     string
	NewState        LocalStateType
	ExpectedState   LocalStateType
	ExpectedVersion int
	Comment         string
}

// ExecutionStore A metadata store of job executions handled by the current compute node
type ExecutionStore interface {
	// GetExecution returns the execution for a given id
	GetExecution(ctx context.Context, id string) (LocalState, error)
	// GetExecutions returns all the executions for a given job
	GetExecutions(ctx context.Context, jobID string) ([]LocalState, error)
	// GetExecutionHistory returns the history of an execution
	GetExecutionHistory(ctx context.Context, id string) ([]LocalStateHistory, error)
	// CreateExecution creates a new execution for a given job
	CreateExecution(ctx context.Context, execution LocalState) error
	// UpdateExecutionState updates the execution state
	UpdateExecutionState(ctx context.Context, request UpdateExecutionStateRequest) error
	// DeleteExecution deletes an execution
	DeleteExecution(ctx context.Context, id string) error
	// GetExecutionCount returns a count of all executions that are in the specified
	// state
	GetExecutionCount(ctx context.Context, state LocalStateType) (uint64, error)
	// Close provides the opportunity for the underlying store to cleanup
	// any resources as the compute node is shutting down
	Close(ctx context.Context) error
}
