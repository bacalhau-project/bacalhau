package orchestrator

import (
	"context"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
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
		evaluationReceipt, err := w.dequeueEvaluation(ctx)
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
		if ack {
			w.ackEvaluation(ctx, evaluationReceipt, ack)
		} else {
			w.nackEvaluation(ctx, evaluationReceipt, ack)
		}
	}
}

// dequeueEvaluation dequeues an evaluation.
func (w *Worker) dequeueEvaluation(ctx context.Context) (*models.EvaluationReceipt, error) {
	recorder := telemetry.NewMetricRecorder()
	defer recorder.RecordFault(ctx, WorkerDequeueFaults)

	evaluation, receiptHandle, err :=
		w.evaluationBroker.Dequeue(w.schedulerProvider.EnabledSchedulers(), w.dequeueTimeout)

	if err != nil {
		return nil, err
	}
	recorder.Success()
	if evaluation == nil {
		return nil, nil
	}

	return &models.EvaluationReceipt{
		Evaluation:    evaluation,
		ReceiptHandle: receiptHandle,
	}, nil
}

// processEvaluation processes an evaluation and returns true if it was processed successfully, false otherwise.
func (w *Worker) processEvaluation(ctx context.Context, evaluation *models.Evaluation) (ack bool) {
	tracker := telemetry.NewMetricRecorder()
	defer tracker.RecordFault(ctx, WorkerProcessFaults, EvalTypeAttribute(evaluation.Type))

	// Check if worker is shutting down while dequeueing
	if w.isShuttingDown(ctx) {
		log.Warn().Msgf("Worker is shutting down, not scheduling evaluation %s", evaluation.ID)
		tracker.Success() // don't treat this as a failure
		return
	}

	// Schedule the evaluation
	scheduler, err := w.schedulerProvider.Scheduler(evaluation.Type)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to retrieve scheduler for evaluation %s", evaluation.ID)
		return
	}
	if err = scheduler.Process(ctx, evaluation); err != nil {
		log.Error().Err(err).Msgf("Failed to process evaluation %s", evaluation.ID)
		return
	}
	ack = true
	tracker.Success()
	return
}

func (w *Worker) ackEvaluation(ctx context.Context, evalReceipt *models.EvaluationReceipt, ack bool) {
	recorder := telemetry.NewMetricRecorder()
	defer recorder.RecordFault(ctx, WorkerAckFaults)

	err := w.evaluationBroker.Ack(evalReceipt.Evaluation.ID, evalReceipt.ReceiptHandle)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to ack evaluation %s", evalReceipt.Evaluation.ID)
	} else {
		recorder.Success()
	}
}

func (w *Worker) nackEvaluation(ctx context.Context, evalReceipt *models.EvaluationReceipt, ack bool) {
	recorder := telemetry.NewMetricRecorder()
	defer recorder.RecordFault(ctx, WorkerNackFaults)

	err := w.evaluationBroker.Nack(evalReceipt.Evaluation.ID, evalReceipt.ReceiptHandle)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to nack evaluation %s", evalReceipt.Evaluation.ID)
	} else {
		recorder.Success()
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
