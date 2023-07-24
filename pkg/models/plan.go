package models

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type PlanExecutionDesiredUpdate struct {
	Execution    *model.ExecutionState
	DesiredState model.ExecutionDesiredState
	Comment      string
}

// Plan holds actions as a result of processing an evaluation by the scheduler.
type Plan struct {
	EvalID      string
	EvalReceipt string
	// TODO: passing the evalID should be enough once we persist evaluations
	Eval     *Evaluation
	Priority int

	Job             *model.Job
	JobStateVersion int

	DesiredJobState model.JobStateType
	Comment         string

	// NewExecutions holds the executions to be created.
	NewExecutions []*model.ExecutionState

	UpdatedExecutions map[model.ExecutionID]*PlanExecutionDesiredUpdate
}

// NewPlan creates a new Plan instance.
func NewPlan(eval *Evaluation, job *model.Job, jobStateVersion int) *Plan {
	return &Plan{
		EvalID:            eval.ID,
		Priority:          eval.Priority,
		Eval:              eval,
		Job:               job,
		JobStateVersion:   jobStateVersion,
		NewExecutions:     []*model.ExecutionState{},
		UpdatedExecutions: make(map[model.ExecutionID]*PlanExecutionDesiredUpdate),
	}
}

// AppendExecution appends the execution to the plan executions.
func (p *Plan) AppendExecution(execution *model.ExecutionState) {
	p.NewExecutions = append(p.NewExecutions, execution)
}

// AppendStoppedExecution marks an execution to be stopped.
func (p *Plan) AppendStoppedExecution(execution *model.ExecutionState, comment string) {
	updateRequest := &PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: model.ExecutionDesiredStateStopped,
		Comment:      comment,
	}
	p.UpdatedExecutions[execution.ID()] = updateRequest
}

// AppendApprovedExecution marks an execution as accepted and ready to be started.
func (p *Plan) AppendApprovedExecution(execution *model.ExecutionState) {
	updateRequest := &PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: model.ExecutionDesiredStateRunning,
	}
	p.UpdatedExecutions[execution.ID()] = updateRequest
}

func (p *Plan) MarkJobCompleted() {
	p.DesiredJobState = model.JobStateCompleted
	p.NewExecutions = []*model.ExecutionState{}
}

func (p *Plan) MarkJobFailed(comment string) {
	p.DesiredJobState = model.JobStateError
	p.Comment = comment

	p.NewExecutions = []*model.ExecutionState{}
	// drop any update that is not stopping an execution
	for id, update := range p.UpdatedExecutions {
		if update.DesiredState != model.ExecutionDesiredStateStopped {
			delete(p.UpdatedExecutions, id)
		}
	}
}
