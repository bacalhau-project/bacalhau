package models

type PlanExecutionUpdate struct {
	Execution    *Execution                `json:"Execution"`
	DesiredState ExecutionDesiredStateType `json:"DesiredState"`
	ComputeState ExecutionStateType        `json:"ComputeState"`
	Event        Event                     `json:"Event"`
}

// Plan holds actions as a result of processing an evaluation by the scheduler.
type Plan struct {
	EvalID      string      `json:"EvalID"`
	EvalReceipt string      `json:"EvalReceipt"`
	Eval        *Evaluation `json:"Eval,omitempty"`
	Priority    int         `json:"Priority"`

	Job *Job `json:"Job,omitempty"`

	DesiredJobState JobStateType `json:"DesiredJobState,omitempty"`
	UpdateMessage   string       `json:"Message,omitempty"`

	// NewExecutions holds the executions to be created.
	NewExecutions []*Execution `json:"NewExecutions,omitempty"`

	UpdatedExecutions map[string]*PlanExecutionUpdate `json:"UpdatedExecutions,omitempty"`

	// NewEvaluations holds the evaluations to be created, such as delayed evaluations when no nodes are available.
	NewEvaluations []*Evaluation `json:"NewEvaluations,omitempty"`

	JobEvents       []Event            `json:"JobEvents,omitempty"`
	ExecutionEvents map[string][]Event `json:"ExecutionEvents,omitempty"`
}

// NewPlan creates a new Plan instance.
func NewPlan(eval *Evaluation, job *Job) *Plan {
	return &Plan{
		EvalID:            eval.ID,
		Priority:          eval.Priority,
		Eval:              eval,
		Job:               job,
		NewExecutions:     []*Execution{},
		UpdatedExecutions: make(map[string]*PlanExecutionUpdate),
		NewEvaluations:    []*Evaluation{},
		JobEvents:         []Event{},
		ExecutionEvents:   make(map[string][]Event),
	}
}

// AppendExecution appends the execution to the plan executions.
func (p *Plan) AppendExecution(execution *Execution, event Event) {
	p.NewExecutions = append(p.NewExecutions, execution)
	p.AppendExecutionEvent(execution.ID, event)
}

// AppendStoppedExecution marks an execution to be stopped.
func (p *Plan) AppendStoppedExecution(execution *Execution, event Event, computeState ExecutionStateType) {
	updateRequest := &PlanExecutionUpdate{
		Execution:    execution,
		DesiredState: ExecutionDesiredStateStopped,
		ComputeState: computeState,
		Event:        event,
	}
	p.UpdatedExecutions[execution.ID] = updateRequest
	p.AppendExecutionEvent(execution.ID, event)
}

// AppendApprovedExecution marks an execution as accepted and ready to be started.
func (p *Plan) AppendApprovedExecution(execution *Execution, event Event) {
	updateRequest := &PlanExecutionUpdate{
		Execution:    execution,
		DesiredState: ExecutionDesiredStateRunning,
		ComputeState: ExecutionStateBidAccepted,
		Event:        event,
	}
	p.UpdatedExecutions[execution.ID] = updateRequest
	p.AppendExecutionEvent(execution.ID, event)
}

// AppendEvaluation appends the evaluation to the plan evaluations.
func (p *Plan) AppendEvaluation(eval *Evaluation) {
	p.NewEvaluations = append(p.NewEvaluations, eval)
}

func (p *Plan) MarkJobCompleted(event Event) {
	p.DesiredJobState = JobStateTypeCompleted
	p.NewExecutions = []*Execution{}
	p.AppendJobEvent(event)
}

// MarkJobRunningIfEligible updates the job state to "Running" under certain conditions.
func (p *Plan) MarkJobRunningIfEligible(event Event) bool {
	// Exit the function if DesiredJobState is already defined.
	if !p.DesiredJobState.IsUndefined() {
		return false
	}

	// Only proceed if the current job state is "Pending" or "Queued".
	if p.Job.State.StateType != JobStateTypePending && p.Job.State.StateType != JobStateTypeQueued {
		return false
	}

	// Check if there are any running executions; if not, exit the function.
	if !p.hasRunningExecutions() {
		return false
	}

	// All conditions met, set DesiredJobState to "Running".
	p.DesiredJobState = JobStateTypeRunning
	p.AppendJobEvent(event)
	return true
}

// MarkJobQueued marks the job as pending.
func (p *Plan) MarkJobQueued(event Event) {
	p.DesiredJobState = JobStateTypeQueued
	p.UpdateMessage = event.Message
	p.AppendJobEvent(event)
}

func (p *Plan) MarkJobFailed(event Event) {
	p.DesiredJobState = JobStateTypeFailed
	p.UpdateMessage = event.Message
	p.AppendJobEvent(event)

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

// AppendJobEvent appends the event to the job events.
func (p *Plan) AppendJobEvent(event Event) {
	p.JobEvents = append(p.JobEvents, event)
}

// AppendExecutionEvent appends the event to the execution events.
func (p *Plan) AppendExecutionEvent(executionID string, event Event) {
	if _, ok := p.ExecutionEvents[executionID]; !ok {
		p.ExecutionEvents[executionID] = []Event{}
	}
	p.ExecutionEvents[executionID] = append(p.ExecutionEvents[executionID], event)
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

// HasPendingWork returns true if the plan has pending work (executions or evaluations)
func (p *Plan) HasPendingWork() bool {
	return len(p.NewExecutions) > 0 || len(p.NewEvaluations) > 0
}
