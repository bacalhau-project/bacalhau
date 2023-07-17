package models

import (
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// Plan holds actions as a result of processing an evaluation by the scheduler.
type Plan struct {
	EvalID      string
	EvalReceipt string
	Priority    int

	// NewExecutions holds the executions to be created.
	NewExecutions []*model.ExecutionState

	// StoppedExecutions holds the executions to be stopped.
	StoppedExecutions []*jobstore.UpdateExecutionRequest
}

// NewPlan creates a new Plan instance.
func NewPlan(eval *Evaluation) *Plan {
	return &Plan{
		EvalID:            eval.ID,
		Priority:          eval.Priority,
		NewExecutions:     []*model.ExecutionState{},
		StoppedExecutions: []*jobstore.UpdateExecutionRequest{},
	}
}

// AppendExecution appends the execution to the plan executions.
func (p *Plan) AppendExecution(execution *model.ExecutionState) {
	p.NewExecutions = append(p.NewExecutions, execution)
}

// AppendStoppedExecution marks an execution to be stopped.
func (p *Plan) AppendStoppedExecution(execution *model.ExecutionState, newState model.ExecutionStateType, comment string) {
	updateRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: execution.ID(),
		NewValues: model.ExecutionState{
			State: newState,
		},
		Comment: comment,
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedState: execution.State,
		},
	}
	p.StoppedExecutions = append(p.StoppedExecutions, &updateRequest)
}

// StoppedExecutionsCount returns the number of stopped executions in the plan.
func (p *Plan) StoppedExecutionsCount() int {
	return len(p.StoppedExecutions)
}

// IsExecutionStopped returns true if the execution is marked to be stopped.
func (p *Plan) IsExecutionStopped(execution *model.ExecutionState) bool {
	for _, stoppedExecution := range p.StoppedExecutions {
		if stoppedExecution.ExecutionID == execution.ID() {
			return true
		}
	}
	return false
}
