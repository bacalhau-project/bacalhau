package orchestrator

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// EvaluationBroker is used to manage brokering of evaluations. When an evaluation is
// created, due to a change in a job specification or a node, we put it into the
// broker. The broker sorts by evaluations by priority and job type. This
// allows us to dequeue the highest priority work first, while also allowing sub-schedulers
// to only dequeue work they know how to handle.
//
// The broker must provide at-least-once delivery semantics. It relies on explicit
// Ack/Nack messages to handle this. If a delivery is not Ack'd in a sufficient time
// span, it will be assumed Nack'd.
//
// The broker must also make sure there is a single inflight evaluation per job, and that
// multiple enqueued evaluations for the same job can be represented as a single most recent evaluation,
type EvaluationBroker interface {
	// Enqueue adds an evaluation to the broker
	// - If the evaluation is already in the broker, it will do nothing
	// - If another evaluation with the same job ID is in the broker, it will not make the new eval
	//   visible until the active eval is Ack'd.
	// - If the evaluation has a WaitUntil time, it will not be visible until that time has passed.
	// - Otherwise the evaluation will be visible to dequeue immediately
	Enqueue(evaluation *model.Evaluation) error

	// EnqueueAll is used to enqueue many evaluations. The map allows evaluations
	// that are being re-enqueued to include their receipt handle.
	// If the evaluation is already in the broker, in flight, and with matching receipt handle, it will
	// re-enqueue the evaluation to be processed again after the previous one is Ack'd.
	EnqueueAll(evaluation map[*model.Evaluation]string) error

	// Dequeue is used to perform a blocking dequeue. The next available evaluation
	// is returned as well as a unique receiptHandle identifier for this dequeue.
	// The receipt handle changes every time the same evaluation is dequeued, such as
	// after a Nack, timeout, state restore or possibly broker lease change.
	// This ensures that previous inflight Dequeue cannot conflict with a
	// Dequeue of the same evaluation after the state change..
	Dequeue(types []string, timeout time.Duration) (*model.Evaluation, string, error)

	// Inflight checks if an EvalID has been delivered but not acknowledged
	// and returns the associated receipt handle for the evaluation.
	Inflight(evaluationID string) (string, bool)

	// InflightExtend resets the Nack timer for the evaluationID if the
	// receipt handle matches and the eval is inflight
	InflightExtend(evaluationID, receiptHandle string) error

	// Ack is used to acknowledge a successful evaluation.
	// The evaluation will be removed from the broker.
	Ack(evalID string, receiptHandle string) error

	// Nack is used to negatively acknowledge an evaluation.
	// The evaluation can be re-enqueued to be processed again
	// without having to wait for the dequeue visibility timeout.
	Nack(evalID string, receiptHandle string) error
}
