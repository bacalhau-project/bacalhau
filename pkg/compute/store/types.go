//go:generate mockgen --source types.go --destination mocks.go --package store
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Execution struct {
	ID              string
	Job             model.Job
	RequesterNodeID string
	ResourceUsage   model.ResourceUsageData
	State           ExecutionState
	Version         int
	CreateTime      time.Time
	UpdateTime      time.Time
	LatestComment   string
}

func NewExecution(
	id string,
	job model.Job,
	requesterNodeID string,
	resourceUsage model.ResourceUsageData) *Execution {
	return &Execution{
		ID:              id,
		Job:             job,
		RequesterNodeID: requesterNodeID,
		ResourceUsage:   resourceUsage,
		State:           ExecutionStateCreated,
		Version:         1,
		CreateTime:      time.Now().UTC(),
		UpdateTime:      time.Now().UTC(),
	}
}

// string returns a string representation of the execution
func (e Execution) String() string {
	return fmt.Sprintf("{ID: %s, Job: %s}", e.ID, e.Job.Metadata.ID)
}

type ExecutionHistory struct {
	ExecutionID   string
	PreviousState ExecutionState
	NewState      ExecutionState
	NewVersion    int
	Comment       string
	Time          time.Time
}

// Summary of an execution that is used in logging and debugging.
type ExecutionSummary struct {
	ExecutionID   string                  `json:"ExecutionID"`
	JobID         string                  `json:"JobID"`
	State         string                  `json:"State"`
	ResourceUsage model.ResourceUsageData `json:"ResourceUsage"`
}

// NewExecutionSummary generate a summary from an execution
func NewExecutionSummary(execution Execution) ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:   execution.ID,
		JobID:         execution.Job.Metadata.ID,
		State:         execution.State.String(),
		ResourceUsage: execution.ResourceUsage,
	}
}

type UpdateExecutionStateRequest struct {
	ExecutionID     string
	NewState        ExecutionState
	ExpectedState   ExecutionState
	ExpectedVersion int
	Comment         string
}

// ExecutionStore A metadata store of job executions handled by the current compute node
type ExecutionStore interface {
	// GetExecution returns the execution for a given id
	GetExecution(ctx context.Context, id string) (Execution, error)
	// GetExecutions returns all the executions for a given job
	GetExecutions(ctx context.Context, jobID string) ([]Execution, error)
	// GetExecutionHistory returns the history of an execution
	GetExecutionHistory(ctx context.Context, id string) ([]ExecutionHistory, error)
	// CreateExecution creates a new execution for a given job
	CreateExecution(ctx context.Context, execution Execution) error
	// UpdateExecutionState updates the execution state
	UpdateExecutionState(ctx context.Context, request UpdateExecutionStateRequest) error
	// DeleteExecution deletes an execution
	DeleteExecution(ctx context.Context, id string) error
	// GetExecutionCount returns a count of all executions that are in the specified
	// state
	GetExecutionCount(ctx context.Context, state ExecutionState) (uint64, error)
	// Close provides the opportunity for the underlying store to cleanup
	// any resources as the compute node is shutting down
	Close(ctx context.Context) error
}
