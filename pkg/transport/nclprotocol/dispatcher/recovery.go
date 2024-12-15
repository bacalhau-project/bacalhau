package dispatcher

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/backoff"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

// recovery manages the recovery process when publish failures occur
type recovery struct {
	mu sync.RWMutex

	// Dependencies
	publisher ncl.OrderedPublisher
	watcher   watcher.Watcher
	state     *dispatcherState
	backoff   backoff.Backoff

	// State
	isRecovering bool      // true while in recovery process
	lastFailure  time.Time // time of most recent failure
	failures     int       // failure count for backoff
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func newRecovery(
	publisher ncl.OrderedPublisher, watcher watcher.Watcher, state *dispatcherState, config Config) *recovery {
	return &recovery{
		publisher: publisher,
		watcher:   watcher,
		state:     state,
		backoff:   backoff.NewExponential(config.BaseRetryInterval, config.MaxRetryInterval),
		stopCh:    make(chan struct{}),
	}
}

// handleError processes a publish error and triggers recovery if needed
func (r *recovery) handleError(ctx context.Context, msg *pendingMessage, err error) {
	// Take recovery lock to handle potential concurrent failures
	r.mu.Lock()
	defer r.mu.Unlock()

	// If we're already recovering, just log and return
	if r.isRecovering {
		log.Ctx(ctx).Trace().
			Err(err).
			EmbedObject(msg).
			Int("failures", r.failures).
			Msg("Additional failure while already recovering")
		return
	}

	// Log error
	log.Ctx(ctx).Error().Err(err).
		EmbedObject(msg).
		Int("failures", r.failures).
		Msg("Failed to publish message")

	// Increment failure count - only once per recovery cycle
	r.failures++
	r.lastFailure = time.Now()

	// Set recovery state
	r.isRecovering = true

	// Stop watcher
	r.watcher.Stop(ctx)
	log.Ctx(ctx).Debug().Msg("Stopped watcher after publish failure")

	// Reset publisher
	r.publisher.Reset(ctx)
	log.Ctx(ctx).Debug().Msg("Reset publisher after publish failure")

	// Reset state - we'll rebuild from checkpoint
	r.state.reset()
	log.Ctx(ctx).Debug().Msg("Reset dispatcher state state after publish failure")

	// Launch recovery goroutine
	r.wg.Add(1)
	go r.recoveryLoop(ctx, r.failures)
}

// recoveryLoop handles the recovery process with backoff
func (r *recovery) recoveryLoop(ctx context.Context, failures int) {
	defer r.wg.Done()
	defer func() {
		r.mu.Lock()
		r.isRecovering = false
		r.mu.Unlock()
	}()

	for {
		// Perform backoff
		backoffDuration := r.backoff.BackoffDuration(failures)
		log.Debug().Int("failures", failures).Dur("backoff", backoffDuration).Msg("Performing backoff")

		// Perform backoff with interruptibility
		timer := time.NewTimer(backoffDuration)
		select {
		case <-timer.C:
		case <-r.stopCh:
			timer.Stop()
			return
		case <-ctx.Done():
			timer.Stop()
			return
		}

		// Just restart the watcher - it will resume from last checkpoint
		if err := r.watcher.Start(ctx); err != nil {
			if r.watcher.Stats().State == watcher.StateRunning {
				log.Debug().Msg("Watcher already after recovery. Exiting recovery loop.")
				return
			}
			select {
			case <-r.stopCh:
				return
			case <-ctx.Done():
				return
			default:
				log.Error().Err(err).Msg("Failed to restart watcher after backoff. Retrying...")
				failures++
			}
		} else {
			log.Debug().Msg("Successfully restarted watcher after backoff")
			return
		}
	}
}

// reset resets the recovery state
func (r *recovery) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.isRecovering = false
	r.lastFailure = time.Time{}
	r.failures = 0
	r.stopCh = make(chan struct{})
}

// getState returns current recovery state
//
//nolint:unused
func (r *recovery) getState() (bool, time.Time, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRecovering, r.lastFailure, r.failures
}

func (r *recovery) stop() {
	r.mu.Lock()
	// Try to close the channel only if it's not already closed
	select {
	case <-r.stopCh:
	default:
		close(r.stopCh)
	}
	r.mu.Unlock()

	// Wait for recovery loop to exit, if running
	r.wg.Wait()
}
