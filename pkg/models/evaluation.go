package models

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
)

const (
	EvalStatusBlocked   = "blocked"
	EvalStatusPending   = "pending"
	EvalStatusComplete  = "complete"
	EvalStatusFailed    = "failed"
	EvalStatusCancelled = "canceled"
)

const (
	EvalTriggerJobRegister = "job-register"
	EvalTriggerJobCancel   = "job-cancel"
	EvalTriggerJobRerun    = "job-rerun"
	EvalTriggerJobUpdate   = "job-update"
	EvalTriggerJobQueue    = "job-queue"
	EvalTriggerJobTimeout  = "job-timeout"

	EvalTriggerExecFailure    = "exec-failure"
	EvalTriggerExecUpdate     = "exec-update"
	EvalTriggerExecTimeout    = "exec-timeout"
	EvalTriggerExecutionLimit = "exec-limit"
	EvalTriggerNodeJoin       = "node-join"
	EvalTriggerNodeLeave      = "node-leave"
)

// Evaluation is just to ask the scheduler to reassess if additional job instances must be
// scheduled or if existing ones must be stopped.
// It is possible that no action is required if the scheduler sees the desired job state matches the observed state.
// This allows the triggers (e.g. APIs, Node Manager) to be lightweight and submit evaluation on state changes without
// having to do complex logic to decide what actions to take.
type Evaluation struct {
	// ID is the unique identifier of the evaluation.
	ID string `json:"ID"`

	// Namespace is the namespace the evaluation is created in
	Namespace string `json:"Namespace"`

	// JobID is the unique identifier of the job.
	JobID string `json:"JobID"`

	// TriggeredBy is the root cause that triggered the evaluation.
	TriggeredBy string `json:"TriggeredBy"`

	// Priority is the priority of the evaluation.
	// e.g. 50 is higher priority than 10, and so will be evaluated first.
	Priority int `json:"Priority"`

	// Type is the type of the job that needs to be evaluated.
	Type string `json:"Type"`

	// Status is the current status of the evaluation.
	Status string `json:"Status"`

	// Comment is to provide additional information about the evaluation.
	Comment string `json:"Comment"`

	// WaitUntil is the time until which the evaluation should be ignored, such as to implement backoff when
	// repeatedly failing to assess a job.
	WaitUntil time.Time `json:"WaitUntil"`

	CreateTime int64 `json:"CreateTime"`
	ModifyTime int64 `json:"ModifyTime"`
}

// NewEvaluation creates a new Evaluation.
func NewEvaluation() *Evaluation {
	now := time.Now().UTC().UnixNano()
	return &Evaluation{
		ID:         idgen.NewEvaluationID(),
		Status:     EvalStatusPending,
		CreateTime: now,
		ModifyTime: now,
	}
}

// WithJobID sets the JobID of the Evaluation.
func (e *Evaluation) WithJobID(jobID string) *Evaluation {
	e.JobID = jobID
	return e
}

// WithJob sets the JobID, Type, Priority nd Namespace of the Evaluation.
func (e *Evaluation) WithJob(job *Job) *Evaluation {
	return e.WithJobID(job.ID).
		WithType(job.Type).
		WithNamespace(job.Namespace).
		WithPriority(job.Priority)
}

// WithNamespace sets the Namespace of the Evaluation.
func (e *Evaluation) WithNamespace(namespace string) *Evaluation {
	e.Namespace = namespace
	return e
}

// WithTriggeredBy sets the TriggeredBy of the Evaluation.
func (e *Evaluation) WithTriggeredBy(triggeredBy string) *Evaluation {
	e.TriggeredBy = triggeredBy
	return e
}

// WithPriority sets the Priority of the Evaluation.
func (e *Evaluation) WithPriority(priority int) *Evaluation {
	e.Priority = priority
	return e
}

// WithType sets the Type of the Evaluation.
func (e *Evaluation) WithType(jobType string) *Evaluation {
	e.Type = jobType
	return e
}

// WithStatus sets the Status of the Evaluation.
func (e *Evaluation) WithStatus(status string) *Evaluation {
	e.Status = status
	return e
}

// WithComment sets the Comment of the Evaluation.
func (e *Evaluation) WithComment(comment string) *Evaluation {
	e.Comment = comment
	return e
}

// WithWaitUntil sets the WaitUntil of the Evaluation.
func (e *Evaluation) WithWaitUntil(waitUntil time.Time) *Evaluation {
	e.WaitUntil = waitUntil
	return e
}

// NewDelayedEvaluation creates a new Evaluation from current one with a WaitUntil time.
func (e *Evaluation) NewDelayedEvaluation(waitUntil time.Time) *Evaluation {
	return &Evaluation{
		ID:          idgen.NewEvaluationID(),
		Namespace:   e.Namespace,
		JobID:       e.JobID,
		TriggeredBy: e.TriggeredBy,
		Priority:    e.Priority,
		Type:        e.Type,
		WaitUntil:   waitUntil,
		Status:      EvalStatusPending,
		CreateTime:  time.Now().UTC().UnixNano(),
		ModifyTime:  time.Now().UTC().UnixNano(),
	}
}

// Normalize ensures that the Evaluation is in a valid state.
func (e *Evaluation) Normalize() *Evaluation {
	if e.ID == "" {
		e.ID = idgen.NewEvaluationID()
	}
	if e.Status == "" {
		e.Status = EvalStatusPending
	}
	if e.CreateTime == 0 {
		e.CreateTime = time.Now().UTC().UnixNano()
	}
	if e.ModifyTime == 0 {
		e.ModifyTime = time.Now().UTC().UnixNano()
	}
	return e
}

// TerminalStatus returns if the current status is terminal and
// will no longer transition.
func (e *Evaluation) TerminalStatus() bool {
	switch e.Status {
	case EvalStatusComplete, EvalStatusFailed, EvalStatusCancelled:
		return true
	default:
		return false
	}
}

func (e *Evaluation) String() string {
	return fmt.Sprintf("<Evaluation %q JobID: %q Namespace: %q>", e.ID, e.JobID, e.Namespace)
}

// ShouldEnqueue checks if a given Evaluation should be enqueued into the
// evaluation_broker
func (e *Evaluation) ShouldEnqueue() bool {
	switch e.Status {
	case EvalStatusPending:
		return true
	case EvalStatusComplete, EvalStatusFailed, EvalStatusBlocked, EvalStatusCancelled:
		return false
	default:
		panic(fmt.Sprintf("unhandled Evaluation (%s) status %s", e.ID, e.Status))
	}
}

// UpdateModifyTime makes sure that time always moves forward, taking into account that different
// server clocks can drift apart.
func (e *Evaluation) UpdateModifyTime() {
	e.ModifyTime = math.Max(time.Now().UTC().UnixNano(), e.CreateTime+1, e.ModifyTime+1)
}

func (e *Evaluation) Copy() *Evaluation {
	if e == nil {
		return nil
	}
	ne := new(Evaluation)
	*ne = *e
	return ne
}

// EvaluationReceipt is a pair of an Evaluation and its ReceiptHandle.
type EvaluationReceipt struct {
	Evaluation *Evaluation `json:"Evaluation"`
	// ReceiptHandle is a unique identifier when dequeue an Evaluation from a broker.
	ReceiptHandle string `json:"ReceiptHandle"`
}
