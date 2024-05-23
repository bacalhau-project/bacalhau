package evaluation

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/rs/zerolog/log"
)

// Watcher is the integration between the jobstore and the evaluation broker.
// It watches for changes in the jobstore and enqueues evaluations in the broker
// It also handles populating the evaluation broker with non-terminal evaluations during startup
type Watcher struct {
	// store is the jobstore to watch
	store jobstore.Store
	// broker is the evaluation broker to enqueue evaluations
	broker orchestrator.EvaluationBroker

	watching  bool
	startOnce sync.Once
	stopOnce  sync.Once
	stopChan  chan struct{}
}

// NewWatcher creates a new Watcher
func NewWatcher(store jobstore.Store, broker orchestrator.EvaluationBroker) *Watcher {
	return &Watcher{
		store:    store,
		broker:   broker,
		stopChan: make(chan struct{}),
	}
}

// IsWatching returns true if the watcher is currently watching
func (w *Watcher) IsWatching() bool {
	return w.watching
}

// Start starts the watcher in a goroutine
func (w *Watcher) Start(ctx context.Context) {
	w.startOnce.Do(func() {
		go w.watchAndEnqueue(ctx)
	})
}

// Backfill populates the broker with non-terminal evaluations
func (w *Watcher) Backfill(ctx context.Context) error {
	// TODO: Implement Backfill
	return nil
}

func (w *Watcher) watchAndEnqueue(ctx context.Context) {
	watcher := w.store.Watch(ctx, jobstore.EvaluationWatcher, jobstore.CreateEvent)
	defer watcher.Close()

	w.watching = true
	defer func() {
		w.watching = false
	}()

	for {
		select {
		case evt := <-watcher.Channel():
			if evt == nil {
				log.Ctx(ctx).Debug().Msg("Watcher channel closed, stopping eval watcher")
				return
			}
			eval, ok := evt.Object.(models.Evaluation)
			if !ok {
				log.Error().Msgf("Received unexpected object type in eval watcher: %T", evt.Object)
				continue
			}
			if err := w.broker.Enqueue(&eval); err != nil {
				log.Error().Err(err).Msgf("Failed to enqueue %s", eval.String())
			}
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Msg("Context cancelled, stopping eval watcher")
			return
		case <-w.stopChan:
			log.Ctx(ctx).Debug().Msg("Stop channel closed, stopping eval watcher")
			return
		}
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})
}
