package requester

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/workerpool"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

const (
	EvalQueueDefaultInputChannelSize = 16
	EvalQueueDefaultWorkerCount      = 16
	DurationOneSecond                = 1 * time.Second
)

type EvaluationQueue struct {
	workers      *workerpool.WorkerPool[jobstore.WatchEvent]
	wg           sync.WaitGroup
	readChannel  chan jobstore.WatchEvent
	closeChannel chan struct{}
}

func NewEvaluationQueue(ctx context.Context, store jobstore.Store, broker orchestrator.EvaluationBroker) (*EvaluationQueue, error) {
	workers, err := workerpool.NewWorkerPool[jobstore.WatchEvent](
		func(evt jobstore.WatchEvent) error {
			var eval models.Evaluation

			err := json.Unmarshal(evt.Object, &eval)
			if err != nil {
				return fmt.Errorf("failed to unmarshall json from job store: %s", err)
			}

			err = broker.Enqueue(&eval)
			if err != nil {
				return fmt.Errorf("failed to enqueue an evaluation: %s", err)
			}

			log.Ctx(ctx).Debug().Str("EvalID", eval.ID).Str("JobID", eval.JobID).Msg("enqueuing evaluation from jobstore event")

			return nil
		},
		workerpool.WithInputChannelSize(EvalQueueDefaultInputChannelSize),
		workerpool.WithWorkerCount(EvalQueueDefaultWorkerCount),
	)
	if err != nil {
		return nil, err
	}
	workers.Start(ctx)

	watchChannel := store.Watch(ctx, jobstore.EvaluationWatcher, jobstore.CreateEvent)

	return &EvaluationQueue{
		readChannel:  watchChannel,
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
			q.workers.Submit(evt)
		}
	}
}

func (q *EvaluationQueue) Stop() {
	close(q.closeChannel)
	q.wg.Wait()
	_ = q.workers.Shutdown(DurationOneSecond)
}

// MakeEvaluationStateUpdater returns a function used as the callback in the Evaluation Broker,
// when the state of an evaluation has changed.
func MakeEvaluationStateUpdater(store jobstore.Store) func(eval *models.Evaluation) {
	return func(eval *models.Evaluation) {
		// Update evaluation in jobstore
	}
}
