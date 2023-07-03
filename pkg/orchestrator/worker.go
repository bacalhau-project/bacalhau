package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/hashicorp/go-metrics"
	"github.com/rs/zerolog/log"
	"go.uber.org/atomic"
)

const (
	WorkerStatusInit     = "Initialized"
	WorkerStatusStarting = "Starting"
	WorkerStatusRunning  = "Running"
	WorkerStatusStopping = "Stopping"
	WorkerStatusStopped  = "Stopped"
)

type WorkerParams struct {
	// SchedulerProvider is responsible for providing the scheduler instance
	// based on the evaluation type.
	SchedulerProvider SchedulerProvider
	// EvaluationBroker is the broker used for handling evaluations.
	EvaluationBroker EvaluationBroker
	// DequeueTimeout is the maximum duration for dequeueing an evaluation.
	DequeueTimeout time.Duration
	// DequeueFailureBackoff defines the backoff strategy when dequeueing an evaluation fails.
	DequeueFailureBackoff backoff.Backoff
}

// Worker is a long-running process that dequeues evaluations, invokes the scheduler
// to process the evaluation, and acknowledges or rejects the evaluation back to the broker.
// The worker is single-threaded and processes one evaluation at a time. An orchestrator
// can have multiple workers working in parallel.
type Worker struct {
	schedulerProvider     SchedulerProvider
	evaluationBroker      EvaluationBroker
	dequeueTimeout        time.Duration
	dequeueFailureBackoff backoff.Backoff

	status       atomic.String
	startOnce    sync.Once
	shutdownOnce sync.Once
}

// NewWorker returns a new Worker instance.
func NewWorker(params *WorkerParams) *Worker {
	return &Worker{
		schedulerProvider:     params.SchedulerProvider,
		evaluationBroker:      params.EvaluationBroker,
		dequeueTimeout:        params.DequeueTimeout,
		dequeueFailureBackoff: params.DequeueFailureBackoff,
		status:                *atomic.NewString(WorkerStatusInit),
	}
}

// Start triggers the worker to start processing evaluations.
// The worker can only start once, and subsequent calls to Start will be ignored.
func (w *Worker) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		w.setStatus(WorkerStatusStarting)
		go w.run(ctx)
	})
}

// Stop triggers the worker to stop processing evaluations.
// The worker will stop after the in-flight evaluation is processed.
func (w *Worker) Stop() {
	w.shutdownOnce.Do(func() {
		w.setStatus(WorkerStatusStopping)
	})
}

// Status returns the current status of the worker.
func (w *Worker) Status() string {
	return w.status.Load()
}

func (w *Worker) run(ctx context.Context) {
	defer w.setStatus(WorkerStatusStopped)
	w.setStatus(WorkerStatusRunning)

	var dequeueFailures int
	for !w.isShuttingDown(ctx) {
		// Dequeue an evaluation and apply backoff if dequeueing fails
		evaluationReceipt, err := w.dequeueEvaluation()
		if err != nil {
			dequeueFailures++
			w.dequeueFailureBackoff.Backoff(ctx, dequeueFailures)
			continue
		}
		// Reset dequeue failures if dequeueing is successful, even if no evaluation is received.
		dequeueFailures = 0

		// If no evaluation is received, continue to the next iteration.
		if evaluationReceipt == nil {
			continue
		}

		// Process the evaluation
		ack := w.processEvaluation(ctx, evaluationReceipt.Evaluation)

		// ack/nack the evaluation
		w.ackEvaluation(evaluationReceipt, ack)
	}
}

// dequeueEvaluation dequeues an evaluation.
func (w *Worker) dequeueEvaluation() (*model.EvaluationReceipt, error) {
	defer metrics.MeasureSince([]string{"bacalhau", "worker", "dequeue"}, time.Now())

	evaluation, receiptHandle, err :=
		w.evaluationBroker.Dequeue(w.schedulerProvider.EnabledSchedulers(), w.dequeueTimeout)

	if err != nil {
		return nil, err
	}
	if evaluation == nil {
		return nil, nil
	}

	return &model.EvaluationReceipt{
		Evaluation:    evaluation,
		ReceiptHandle: receiptHandle,
	}, nil
}

// processEvaluation processes an evaluation and returns true if it was processed successfully, false otherwise.
func (w *Worker) processEvaluation(ctx context.Context, evaluation *model.Evaluation) (ack bool) {
	defer metrics.MeasureSince([]string{"bacalhau", "worker", "process"}, time.Now())
	// Check if worker is shutting down while dequeueing
	if w.isShuttingDown(ctx) {
		log.Warn().Msgf("Worker is shutting down, not scheduling evaluation %s", evaluation.ID)
		return
	}

	// Schedule the evaluation
	scheduler := w.schedulerProvider.Scheduler(evaluation.Type)
	if scheduler == nil {
		log.Error().Msgf("Failed to retrieve scheduler for evaluation %s of type %s", evaluation.ID, evaluation.Type)
		return
	}
	if err := scheduler.Process(evaluation); err != nil {
		log.Error().Err(err).Msgf("Failed to process evaluation %s", evaluation.ID)
		return
	}
	ack = true
	return
}

// ackEvaluation acks or nacks an evaluation back to the broker.
func (w *Worker) ackEvaluation(evalReceipt *model.EvaluationReceipt, ack bool) {
	operation := "nack"
	if ack {
		operation = "ack"
	}
	defer metrics.MeasureSince([]string{"bacalhau", "worker", operation}, time.Now())

	var err error
	if ack {
		err = w.evaluationBroker.Ack(evalReceipt.Evaluation.ID, evalReceipt.ReceiptHandle)
	} else {
		err = w.evaluationBroker.Nack(evalReceipt.Evaluation.ID, evalReceipt.ReceiptHandle)
	}

	if err != nil {
		log.Error().Err(err).Msgf("Failed to %s evaluation %s", operation, evalReceipt.Evaluation.ID)
	} else {
		log.Debug().Msgf("%sed evaluation %s", operation, evalReceipt.Evaluation.ID)
	}
}

func (w *Worker) setStatus(newStatus string) {
	oldStatus := w.status.Swap(newStatus)
	if oldStatus != newStatus {
		log.Trace().Msgf("Worker status changed from %s to %s", oldStatus, newStatus)
	}
}

// isShuttingDown returns true if the worker is in the process of shutting down or has already shut down.
func (w *Worker) isShuttingDown(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return w.Status() == WorkerStatusStopping || w.Status() == WorkerStatusStopped
	}
}
