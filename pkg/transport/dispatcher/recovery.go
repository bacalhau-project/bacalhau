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
}

func newRecovery(
	publisher ncl.OrderedPublisher, watcher watcher.Watcher, state *dispatcherState, config Config) *recovery {
	return &recovery{
		publisher: publisher,
		watcher:   watcher,
		state:     state,
		backoff:   backoff.NewExponential(config.BaseRetryInterval, config.MaxRetryInterval),
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
	go r.recoveryLoop(ctx, r.failures)
}

// recoveryLoop handles the recovery process with backoff
func (r *recovery) recoveryLoop(ctx context.Context, failures int) {
	defer func() {
		r.mu.Lock()
		r.isRecovering = false
		r.mu.Unlock()
	}()

	for {
		// Perform backoff
		backoffDuration := r.backoff.BackoffDuration(failures)
		log.Debug().Int("failures", failures).Dur("backoff", backoffDuration).Msg("Performing backoff")
		r.backoff.Backoff(ctx, failures)

		// Just restart the watcher - it will resume from last checkpoint
		if err := r.watcher.Start(ctx); err != nil {
			if r.watcher.Stats().State == watcher.StateRunning {
				log.Debug().Msg("Watcher already after recovery. Exiting recovery loop.")
				return
			}
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).Msg("Failed to restart watcher after backoff. Retrying...")
			failures++
		} else {
			log.Info().Msg("Successfully restarted watcher after backoff")
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
}

// getState returns current recovery state
//
//nolint:unused
func (r *recovery) getState() (bool, time.Time, int) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.isRecovering, r.lastFailure, r.failures
}
