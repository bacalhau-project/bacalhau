package models

import (
	"github.com/google/uuid"
	"time"
)

type PlanExecutionDesiredUpdate struct {
	Execution    *Execution                `json:"Execution"`
	DesiredState ExecutionDesiredStateType `json:"DesiredState"`
	Comment      string                    `json:"Comment,omitempty"`
}

// Plan holds actions as a result of processing an evaluation by the scheduler.
type Plan struct {
	EvalID      string `json:"EvalID"`
	EvalReceipt string `json:"EvalReceipt"`
	// TODO: passing the evalID should be enough once we persist evaluations
	Eval     *Evaluation `json:"Eval,omitempty"`
	Priority int         `json:"Priority"`

	Job *Job `json:"Job,omitempty"`

	DesiredJobState JobStateType `json:"DesiredJobState,omitempty"`
	Comment         string       `json:"Comment,omitempty"`

	// NewExecutions holds the executions to be created.
	NewExecutions []*Execution `json:"NewExecutions,omitempty"`

	NewEvaluation *Evaluation `json:"NewEvaluation,omitempty"`

	UpdatedExecutions map[string]*PlanExecutionDesiredUpdate `json:"UpdatedExecutions,omitempty"`
}

// NewPlan creates a new Plan instance.
func NewPlan(eval *Evaluation, job *Job) *Plan {
	return &Plan{
		EvalID:            eval.ID,
		Priority:          eval.Priority,
		Eval:              eval,
		Job:               job,
		NewExecutions:     []*Execution{},
		UpdatedExecutions: make(map[string]*PlanExecutionDesiredUpdate),
	}
}

// AppendExecution appends the execution to the plan executions.
func (p *Plan) AppendExecution(execution *Execution) {
	p.NewExecutions = append(p.NewExecutions, execution)
}

// AppendStoppedExecution marks an execution to be stopped.
func (p *Plan) AppendStoppedExecution(execution *Execution, comment string) {
	updateRequest := &PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: ExecutionDesiredStateStopped,
		Comment:      comment,
	}
	p.UpdatedExecutions[execution.ID] = updateRequest
}

// AppendApprovedExecution marks an execution as accepted and ready to be started.
func (p *Plan) AppendApprovedExecution(execution *Execution) {
	updateRequest := &PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: ExecutionDesiredStateRunning,
	}
	p.UpdatedExecutions[execution.ID] = updateRequest
}

func (p *Plan) MarkJobCompleted() {
	p.DesiredJobState = JobStateTypeCompleted
	p.NewExecutions = []*Execution{}
}

// Defer this job, to be re-evaluated after the given delay
func (p *Plan) DeferEvaluation(delay time.Duration) {
	now := time.Now()
	p.NewEvaluation = &Evaluation{
		ID:          uuid.NewString(),
		JobID:       p.Job.ID,
		TriggeredBy: EvalTriggerDefer,
		Type:        p.Job.Type,
		Status:      EvalStatusPending,
		CreateTime:  now.UTC().UnixNano(),
		ModifyTime:  now.UTC().UnixNano(),
		WaitUntil:   now.Add(delay),
	}
	p.DesiredJobState = JobStateTypePending
}

// MarkJobRunningIfEligible updates the job state to "Running" under certain conditions.
func (p *Plan) MarkJobRunningIfEligible() {
	// Exit the function if DesiredJobState is already defined.
	if !p.DesiredJobState.IsUndefined() {
		return
	}

	// Only proceed if the current job state is "Pending".
	if p.Job.State.StateType != JobStateTypePending {
		return
	}

	// Check if there are any running executions; if not, exit the function.
	if !p.hasRunningExecutions() {
		return
	}

	// All conditions met, set DesiredJobState to "Running".
	p.DesiredJobState = JobStateTypeRunning
}

func (p *Plan) MarkJobFailed(comment string) {
	p.DesiredJobState = JobStateTypeFailed
	p.Comment = comment

	p.NewExecutions = []*Execution{}
	// drop any update that is not stopping an execution
	for id, update := range p.UpdatedExecutions {
		if update.DesiredState != ExecutionDesiredStateStopped {
			delete(p.UpdatedExecutions, id)
		}
	}
}

// hasRunningExecutions returns true if the plan has executions in desired state "Running".
func (p *Plan) hasRunningExecutions() bool {
	for _, exec := range p.NewExecutions {
		if exec.DesiredState.StateType == ExecutionDesiredStateRunning {
			return true
		}
	}
	for _, update := range p.UpdatedExecutions {
		if update.DesiredState == ExecutionDesiredStateRunning {
			return true
		}
	}
	return false
}
