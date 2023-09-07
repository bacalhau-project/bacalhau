package models

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
)

const (
	// Evaluations can be in one of several states, with only one of them `pending`
	// being an active state, the rest being considered terminal. The evaluation can
	// move from `pending` to any other state, but may _only_ move back to pending
	// if it is in the `blocked` state

	// The evaluation's initial state, it will stay in pending until it has been
	// processed by the Evaluation Broker.
	EvalStatusPending = "pending"

	// If an evaluation is unable to be completed, possibly due to a temporal
	// constraint, it will be left in the blocked state.
	EvalStatusBlocked = "blocked"

	// When fully evaluated, the evaluation will be in the completed state.
	EvalStatusComplete = "complete"

	// This status applies when an evaluation fails during procesing
	EvalStatusFailed = "failed"

	// As a result of processing, the evaluation may be determined to now be
	// unnecessary and cancelled as a result, which will leave it in this state.
	EvalStatusCancelled = "canceled"
)

const (
	EvalTriggerJobRegister     = "job-register"
	EvalTriggerJobCancel       = "job-cancel"
	EvalTriggerRetryFailedExec = "exec-failure"
	EvalTriggerExecUpdate      = "exec-update"
)

// Evaluation is just to ask the scheduler to reassess if additional job instances must be
// scheduled or if existing ones must be stopped.
// It is possible that no action is required if the scheduler sees the desired job state matches the observed state.
// This allows the triggers (e.g. APIs, Node Manager) to be lightweight and submit evaluation on state changes without
// having to do complex logic to decide what actions to take.
type Evaluation struct {
	// ID is the unique identifier of the evaluation.
	ID string

	// Namespace is the namespace the evaluation is created in
	Namespace string

	// JobID is the unique identifier of the job.
	JobID string

	// TriggeredBy is the root cause that triggered the evaluation.
	TriggeredBy string

	// Priority is the priority of the evaluation.
	// e.g. 50 is higher priority than 10, and so will be evaluated first.
	Priority int

	// Type is the type of the job that needs to be evaluated.
	Type string

	// Status is the current status of the evaluation.
	Status string

	// Comment is to provide additional information about the evaluation.
	Comment string

	// WaitUntil is the time until which the evaluation should be ignored, such as to implement backoff when
	// repeatedly failing to assess a job.
	WaitUntil time.Time

	CreateTime int64
	ModifyTime int64
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
	Evaluation *Evaluation
	// ReceiptHandle is a unique identifier when dequeue an Evaluation from a broker.
	ReceiptHandle string
}

// EvaluationStateChanged is used as a callback mechanism in cases where
// a component is interested in state changes within an evaluation.
type EvaluationStateChanged func(e *Evaluation)
