package watcher

import (
	"context"
	"errors"
	mathgo "math"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
)

// watcher holds information about a single event watcher
type watcher struct {
	id      string // unique identifier for this watcher
	handler EventHandler
	store   EventStore // event store for fetching events and checkpoints
	options *watchOptions

	nextEventIterator      EventIterator
	lastProcessedSeqNum    uint64
	lastProcessedEventTime time.Time
	lastListenTime         time.Time

	cancel  context.CancelFunc
	stopped chan struct{} // channel to signal that the watcher has stopped
	state   State
	mu      sync.RWMutex
}

// newWatcher creates a new watcher with the given parameters
func newWatcher(ctx context.Context, id string, handler EventHandler, store EventStore, opts ...WatchOption) (*watcher, error) {
	options := defaultWatchOptions()
	for _, opt := range opts {
		opt(options)
	}

	if err := options.validate(); err != nil {
		return nil, NewWatcherError(id, err)
	}

	w := &watcher{
		id:      id,
		handler: handler,
		store:   store,
		options: options,
		state:   StateIdle,
	}

	// Determine the starting iterator
	iterator, err := w.determineStartingIterator(ctx, options.initialEventIterator)
	if err != nil {
		return nil, NewWatcherError(id, err)
	}
	w.nextEventIterator = iterator

	return w, nil
}

func (w *watcher) determineStartingIterator(ctx context.Context, initial EventIterator) (EventIterator, error) {
	// First try to get checkpoint
	checkpoint, err := w.store.GetCheckpoint(ctx, w.id)
	if err == nil {
		return AfterSequenceNumberIterator(checkpoint), nil
	}
	if !errors.Is(err, ErrCheckpointNotFound) {
		return EventIterator{}, err
	}

	// No checkpoint found, handle initial iterator
	if initial.Type == EventIteratorLatest {
		latestSeqNum, err := w.store.GetLatestEventNum(ctx)
		if err != nil {
			return EventIterator{}, err
		}
		return AfterSequenceNumberIterator(latestSeqNum), nil
	}

	return initial, nil
}

// ID returns the unique identifier for the watcher
func (w *watcher) ID() string {
	return w.id
}

// Stats returns the current statistics and state of the watcher
func (w *watcher) Stats() Stats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return Stats{
		ID:                     w.id,
		State:                  w.state,
		NextEventIterator:      w.nextEventIterator,
		LastProcessedSeqNum:    w.lastProcessedSeqNum,
		LastProcessedEventTime: w.lastProcessedEventTime,
		LastListenTime:         w.lastListenTime,
	}
}

// Start begins the event listening process
func (w *watcher) Start() {
	w.mu.Lock()
	if w.state != StateIdle && w.state != StateStopped {
		log.Warn().Str("watcher_id", w.id).Str("state", string(w.state)).
			Msg("watcher already running/stopped, skipping start")
		w.mu.Unlock()
		return
	}

	var ctx context.Context
	ctx, w.cancel = context.WithCancel(context.Background())
	w.stopped = make(chan struct{}, 1)
	w.state = StateRunning
	log.Ctx(ctx).Debug().Str("watcher_id", w.ID()).Str("starting_at", w.nextEventIterator.String()).
		Msg("starting watcher")
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.state = StateStopped
		w.mu.Unlock()
		w.stopped <- struct{}{}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			response, err := w.fetchWithBackoff(ctx)
			if err != nil {
				continue
			}

			for _, event := range response.Events {
				if err = w.processEventWithRetry(ctx, event); err != nil {
					// if the error is due to the context being canceled, return
					if errors.Is(err, context.Canceled) {
						return
					}
					// otherwise, the strategy is to skip and continue processing the next event
					continue
				}
				w.updateLastProcessedEvent(event)
			}
			// update the next event iterator
			w.nextEventIterator = response.NextEventIterator
		}
	}
}

// fetchWithBackoff fetches events with retries and backoff
// no maximum retries are applied here as the watcher should keep trying to fetch events
func (w *watcher) fetchWithBackoff(ctx context.Context) (*GetEventsResponse, error) {
	backoff := w.options.initialBackoff
	for {
		response, err := w.store.GetEvents(ctx, GetEventsRequest{
			EventIterator: w.nextEventIterator,
			Limit:         w.options.batchSize,
			Filter:        w.options.filter,
		})
		if err == nil {
			w.lastListenTime = time.Now()
			return response, nil
		}

		if errors.Is(err, context.Canceled) {
			return nil, err
		}

		log.Error().Err(err).Str("watcher_id", w.id).Msg("failed to fetch events. Retrying...")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff = time.Duration(math.Min(float64(backoff)*2, float64(w.options.maxBackoff)))
		}
	}
}

// processEventWithRetry processes an event with retries
// the number of retries is limited by the maxRetries option before skipping the event,
// or unlimited if the retry strategy is RetryStrategyBlock
func (w *watcher) processEventWithRetry(ctx context.Context, event Event) error {
	backoff := w.options.initialBackoff
	maxRetries := w.options.maxRetries
	if w.options.retryStrategy == RetryStrategyBlock {
		maxRetries = mathgo.MaxInt
	}
	var err error
	for i := 0; i < maxRetries; i++ {
		err = w.handler.HandleEvent(ctx, event)
		if err == nil {
			return nil
		}

		if errors.Is(err, context.Canceled) {
			return err
		}

		log.Error().Err(err).Str("watcher_id", w.id).Uint64("event_seq", event.SeqNum).
			Msg("failed to process event. Retrying...")

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff = time.Duration(math.Min(float64(backoff)*2, float64(w.options.maxBackoff)))
		}
	}
	if err != nil {
		return NewEventHandlingError(w.id, event.SeqNum, err)
	}
	return nil
}

func (w *watcher) updateLastProcessedEvent(event Event) {
	w.lastProcessedSeqNum = event.SeqNum
	w.lastProcessedEventTime = event.Timestamp
}

// Stop gracefully stops the watcher
func (w *watcher) Stop(ctx context.Context) {
	w.mu.Lock()
	if w.state != StateRunning {
		log.Warn().Str("watcher_id", w.id).Str("state", string(w.state)).
			Msg("watcher not running, skipping stop")
		w.mu.Unlock()
		return
	}
	w.state = StateStopping
	w.mu.Unlock()

	log.Ctx(ctx).Debug().Str("watcher_id", w.id).Msg("stopping watcher")
	// stop the watcher
	w.cancel()

	// wait for the watcher to stop
	select {
	case <-w.stopped:
		log.Ctx(ctx).Debug().Str("watcher_id", w.id).Msg("watcher stopped")
	case <-ctx.Done():
		log.Ctx(ctx).Warn().Str("watcher_id", w.id).Msg("watcher stop timed out")
	}
}

// Checkpoint saves the current progress of the watcher
func (w *watcher) Checkpoint(ctx context.Context, eventSeqNum uint64) error {
	return w.store.StoreCheckpoint(ctx, w.id, eventSeqNum)
}

// SeekToOffset moves the watcher to a specific event sequence number
func (w *watcher) SeekToOffset(ctx context.Context, eventSeqNum uint64) error {
	// stop the watcher so that it doesn't process events while we're updating the offset
	w.Stop(ctx)

	// update the offset
	w.nextEventIterator = AfterSequenceNumberIterator(eventSeqNum)

	// persist the offset so that the watcher resumes at the correct position if started
	if err := w.store.StoreCheckpoint(ctx, w.id, eventSeqNum); err != nil {
		log.Ctx(ctx).Error().Err(err).Str("watcher_id", w.id).
			Msg("seek failed to persist offset. Watcher might not resume at the correct position")
	}

	// restart the watcher
	go w.Start()
	return nil
}

// compile time check for interface conformance
var _ Watcher = &watcher{}
