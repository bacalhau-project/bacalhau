package models

type PlanExecutionDesiredUpdate struct {
	Execution    *Execution                `json:"Execution"`
	DesiredState ExecutionDesiredStateType `json:"DesiredState"`
	Event        Event                     `json:"Event"`
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
	Event           Event        `json:"Event,omitempty"`

	// NewExecutions holds the executions to be created.
	NewExecutions []*Execution `json:"NewExecutions,omitempty"`

	UpdatedExecutions map[string]*PlanExecutionDesiredUpdate `json:"UpdatedExecutions,omitempty"`

	// NewEvaluations holds the evaluations to be created, such as delayed evaluations when no nodes are available.
	NewEvaluations []*Evaluation `json:"NewEvaluations,omitempty"`
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
		NewEvaluations:    []*Evaluation{},
	}
}

// AppendExecution appends the execution to the plan executions.
func (p *Plan) AppendExecution(execution *Execution) {
	p.NewExecutions = append(p.NewExecutions, execution)
}

// AppendStoppedExecution marks an execution to be stopped.
func (p *Plan) AppendStoppedExecution(execution *Execution, event Event) {
	updateRequest := &PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: ExecutionDesiredStateStopped,
		Event:        event,
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

// AppendEvaluation appends the evaluation to the plan evaluations.
func (p *Plan) AppendEvaluation(eval *Evaluation) {
	p.NewEvaluations = append(p.NewEvaluations, eval)
}

func (p *Plan) MarkJobCompleted() {
	p.DesiredJobState = JobStateTypeCompleted
	p.NewExecutions = []*Execution{}
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

// MarkJobQueued marks the job as pending.
func (p *Plan) MarkJobQueued(event Event) {
	p.DesiredJobState = JobStateTypeQueued
	p.Event = event
}

func (p *Plan) MarkJobFailed(event Event) {
	p.DesiredJobState = JobStateTypeFailed
	p.Event = event

	p.NewExecutions = []*Execution{}
	// drop any update that is not stopping an execution
	for id, update := range p.UpdatedExecutions {
		if update.DesiredState != ExecutionDesiredStateStopped {
			delete(p.UpdatedExecutions, id)
		}
	}
}

// IsJobFailed returns true if the plan is marking the job as failed
func (p *Plan) IsJobFailed() bool {
	return p.DesiredJobState == JobStateTypeFailed
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
