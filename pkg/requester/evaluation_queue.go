package requester

import (
	"context"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

const (
	EvalQueueDefaultInputChannelSize = 16
	EvalQueueDefaultWorkerCount      = 16
	DurationOneSecond                = 1 * time.Second
)

type EvaluationQueue struct {
	workers      *workerpool.WorkerPool
	wg           sync.WaitGroup
	broker       orchestrator.EvaluationBroker
	readChannel  chan jobstore.WatchEvent
	closeChannel chan struct{}
}

func NewEvaluationQueue(ctx context.Context, store jobstore.Store, broker orchestrator.EvaluationBroker) (*EvaluationQueue, error) {
	workers := workerpool.New(EvalQueueDefaultWorkerCount)
	watchChannel := store.Watch(ctx, jobstore.EvaluationWatcher, jobstore.CreateEvent)

	return &EvaluationQueue{
		readChannel:  watchChannel,
		broker:       broker,
		workers:      workers,
		closeChannel: make(chan struct{}),
	}, nil
}

func (q *EvaluationQueue) Start(ctx context.Context) {
	q.wg.Add(1)
	go q.run(ctx)
}

func (q *EvaluationQueue) run(ctx context.Context) {
	defer q.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-q.closeChannel:
			return
		case evt := <-q.readChannel:
			q.workers.Submit(func() {
				eval := evt.Object.(models.Evaluation)

				err := q.broker.Enqueue(&eval)
				if err != nil {
					log.Ctx(ctx).
						Error().
						Err(err).
						Str("EvalID", eval.ID).
						Str("JobID", eval.JobID).
						Msg("failed to enqueue jobstore event")
				} else {
					log.Ctx(ctx).Debug().
						Str("EvalID", eval.ID).
						Str("JobID", eval.JobID).
						Msg("enqueuing evaluation from jobstore event")
				}
			})
		}
	}
}

func (q *EvaluationQueue) Stop() {
	close(q.closeChannel)
	q.wg.Wait()

	q.workers.Stop()
}

// MakeEvaluationStateUpdater returns a function used as the callback in the Evaluation Broker,
// when the state of an evaluation has changed.
func MakeEvaluationStateUpdater(store jobstore.Store) func(eval *models.Evaluation) {
	return func(eval *models.Evaluation) {
		// Update evaluation in jobstore
	}
}
