//go:generate mockgen --source types.go --destination mocks.go --package store
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type LocalExecutionState struct {
	Execution       *models.Execution
	RequesterNodeID string
	State           LocalExecutionStateType
	Version         int
	CreateTime      time.Time
	UpdateTime      time.Time
	LatestComment   string
}

func NewLocalExecutionState(execution *models.Execution, requesterNodeID string) *LocalExecutionState {
	return &LocalExecutionState{
		Execution:       execution,
		RequesterNodeID: requesterNodeID,
		State:           ExecutionStateCreated,
		Version:         1,
		CreateTime:      time.Now().UTC(),
		UpdateTime:      time.Now().UTC(),
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
	return fmt.Sprintf("{ID: %s, Job: %s}", e.Execution.ID, e.Execution.Job.ID)
}

// ToSummary returns a summary of the execution
func (e *LocalExecutionState) ToSummary() ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:        e.Execution.ID,
		JobID:              e.Execution.Job.ID,
		State:              e.State.String(),
		AllocatedResources: e.Execution.AllocatedResources,
	}
}

type LocalStateHistory struct {
	ExecutionID   string
	PreviousState LocalExecutionStateType
	NewState      LocalExecutionStateType
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
	NewState        LocalExecutionStateType
	ExpectedState   LocalExecutionStateType
	ExpectedVersion int
	Comment         string
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
	// Close provides the opportunity for the underlying store to cleanup
	// any resources as the compute node is shutting down
	Close(ctx context.Context) error
}
