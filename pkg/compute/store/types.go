package store

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Execution struct {
	ID            string
	Shard         model.JobShard
	ResourceUsage model.ResourceUsageData
	State         ExecutionState
	Version       int
	CreateTime    time.Time
	UpdateTime    time.Time
	LatestComment string
}

func NewExecution(id string, shard model.JobShard, resourceUsage model.ResourceUsageData) *Execution {
	return &Execution{
		ID:            id,
		Shard:         shard,
		ResourceUsage: resourceUsage,
		State:         ExecutionStateCreated,
		Version:       1,
		CreateTime:    time.Now(),
		UpdateTime:    time.Now(),
	}
}

// string returns a string representation of the execution
func (e Execution) String() string {
	return fmt.Sprintf("{ID: %s, Shard: %s, State: %s}", e.ID, e.Shard.ID(), e.State)
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
	ShardID       string                  `json:"ShardID"`
	State         string                  `json:"State"`
	ResourceUsage model.ResourceUsageData `json:"ResourceUsage"`
}

// NewExecutionSummary generate a summary from an execution
func NewExecutionSummary(execution Execution) ExecutionSummary {
	return ExecutionSummary{
		ExecutionID:   execution.ID,
		ShardID:       execution.Shard.ID(),
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
	// GetExecutions returns all the executions for a given shard
	GetExecutions(ctx context.Context, sharedID string) ([]Execution, error)
	// GetExecutionHistory returns the history of an execution
	GetExecutionHistory(ctx context.Context, id string) ([]ExecutionHistory, error)
	// CreateExecution creates a new execution for a given shard
	CreateExecution(ctx context.Context, execution Execution) error
	// UpdateExecutionState updates the execution state
	UpdateExecutionState(ctx context.Context, request UpdateExecutionStateRequest) error
	// DeleteExecution deletes an execution
	DeleteExecution(ctx context.Context, id string) error
}
