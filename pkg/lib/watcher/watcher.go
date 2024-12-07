package watcher

import (
	"context"
	"errors"
	"fmt"
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

	nextEventIterator      EventIterator // for processing
	checkpointIterator     EventIterator // for confirmed checkpoints
	lastProcessedSeqNum    uint64
	lastProcessedEventTime time.Time
	lastListenTime         time.Time

	cancel  context.CancelFunc
	stopped chan struct{} // channel to signal that the watcher has stopped
	state   State
	mu      sync.RWMutex
}

// New creates a new watcher with the given parameters
func New(ctx context.Context, id string, store EventStore, opts ...WatchOption) (Watcher, error) {
	options := defaultWatchOptions()
	for _, opt := range opts {
		opt(options)
	}

	if err := options.validate(); err != nil {
		return nil, NewWatcherError(id, err)
	}

	w := &watcher{
		id:      id,
		store:   store,
		options: options,
		state:   StateIdle,
		stopped: make(chan struct{}),
	}

	// Initially close stopped channel since the watcher starts in idle/stopped state
	close(w.stopped)

	// Determine the starting iterator
	iterator, err := w.determineStartingIterator(ctx, options.initialEventIterator)
	if err != nil {
		return nil, NewWatcherError(id, err)
	}
	w.checkpointIterator = iterator
	w.nextEventIterator = iterator

	// set the handler if provided
	if options.handler != nil {
		if err = w.SetHandler(options.handler); err != nil {
			return nil, err
		}
	}

	// Auto-start if requested and handler is set
	if options.autoStart {
		if err = w.Start(ctx); err != nil {
			return nil, NewWatcherError(id, fmt.Errorf("failed to auto-start watcher: %w", err))
		}
	}

	return w, nil
}

func (w *watcher) determineStartingIterator(ctx context.Context, initial EventIterator) (EventIterator, error) {
	// First try to get checkpoint if not ignoring it
	if !w.options.ignoreCheckpoint {
		checkpoint, err := w.store.GetCheckpoint(ctx, w.id)
		if err == nil {
			return AfterSequenceNumberIterator(checkpoint), nil
		}
		if !errors.Is(err, ErrCheckpointNotFound) {
			return EventIterator{}, err
		}
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
		CheckpointIterator:     w.checkpointIterator,
		LastProcessedSeqNum:    w.lastProcessedSeqNum,
		LastProcessedEventTime: w.lastProcessedEventTime,
		LastListenTime:         w.lastListenTime,
	}
}

// SetHandler sets the event handler for this watcher
func (w *watcher) SetHandler(handler EventHandler) error {
	if handler == nil {
		return errors.New("handler cannot be nil")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.handler != nil {
		return ErrHandlerExists
	}

	w.handler = handler
	return nil
}

// Start begins the event listening process
func (w *watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.state != StateIdle && w.state != StateStopped {
		w.mu.Unlock()
		return NewWatcherError(w.id, fmt.Errorf("cannot start watcher in state %s", w.state))
	}

	if w.handler == nil {
		w.mu.Unlock()
		return NewWatcherError(w.id, ErrNoHandler)
	}

	ctx, w.cancel = context.WithCancel(ctx)
	w.stopped = make(chan struct{})
	w.state = StateRunning
	w.nextEventIterator = w.checkpointIterator
	log.Ctx(ctx).Debug().
		Str("watcher_id", w.ID()).
		Str("starting_at", w.nextEventIterator.String()).
		Strs("object_types", w.options.filter.ObjectTypes).
		Msg("starting watcher")
	log.Trace().Msgf("starting watcher %+v", w)
	w.mu.Unlock()

	go w.run(ctx)
	return nil
}

// run is the main event processing loop
func (w *watcher) run(ctx context.Context) {
	defer func() {
		w.mu.Lock()
		w.state = StateStopped
		w.mu.Unlock()
		close(w.stopped)
	}()

	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Debug().Str("watcher_id", w.id).Msg("context canceled. Stopping watcher")
			return
		default:
			response, err := w.fetchWithBackoff(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
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
			WatcherID:     w.ID(),
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
	if w.state == StateStopped || w.state == StateIdle {
		w.state = StateStopped
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
	if err := w.store.StoreCheckpoint(ctx, w.id, eventSeqNum); err != nil {
		return err
	}
	log.Ctx(ctx).Trace().Str("watcher_id", w.id).Uint64("event_seq", eventSeqNum).
		Msg("checkpoint saved")

	// Update checkpoint iterator after successful store
	w.mu.Lock()
	w.checkpointIterator = AfterSequenceNumberIterator(eventSeqNum)
	w.mu.Unlock()

	return nil
}

// SeekToOffset moves the watcher to a specific event sequence number
func (w *watcher) SeekToOffset(ctx context.Context, eventSeqNum uint64) error {
	log.Ctx(ctx).Debug().Str("watcher_id", w.id).Uint64("event_seq", eventSeqNum).
		Msg("seeking to event sequence number")
	// stop the watcher so that it doesn't process events while we're updating the offset
	w.Stop(ctx)

	// persist the offset so that the watcher resumes at the correct position if started
	if err := w.Checkpoint(ctx, eventSeqNum); err != nil {
		return NewCheckpointError(w.id, fmt.Errorf("failed to persist seek offset: %w", err))
	}

	// Restart watcher
	if err := w.Start(ctx); err != nil {
		return NewWatcherError(w.id, fmt.Errorf("failed to restart watcher after seek: %w", err))
	}
	return nil
}

// compile time check for interface conformance
var _ Watcher = &watcher{}
