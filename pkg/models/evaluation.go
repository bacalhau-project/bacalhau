package models

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
)

const (
	EvalStatusBlocked   = "blocked"
	EvalStatusPending   = "pending"
	EvalStatusComplete  = "complete"
	EvalStatusFailed    = "failed"
	EvalStatusCancelled = "canceled"
)

const (
	EvalTriggerJobRegister     = "job-register"
	EvalTriggerJobCancel       = "job-cancel"
	EvalTriggerRetryFailedExec = "exec-failure"
	EvalTriggerExecUpdate      = "exec-update"
	EvalTriggerDefer           = "defer"
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
