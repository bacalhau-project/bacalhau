package dispatcher

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dario.cat/mergo"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
)

// Dispatcher handles reliable delivery of events from a watcher to NATS.
// It maintains sequence ordering, handles retries, and provides checkpointing
// for resuming after restarts.
type Dispatcher struct {
	config  Config
	watcher watcher.Watcher
	mu      sync.RWMutex

	// State tracking
	state    *dispatcherState
	recovery *recovery

	// Channels for shutdown coordination
	running    bool
	stopCh     chan struct{}
	routinesWg sync.WaitGroup
}

// New creates a new Dispatcher with the given configuration and dependencies.
// The provided publisher will be used to publish messages to NATS.
// The watcher provides the source of events.
// The messageCreator determines how events are converted to messages.
// Returns an error if any dependencies are nil or if config validation fails.
func New(publisher ncl.OrderedPublisher,
	watcher watcher.Watcher,
	messageCreator transport.MessageCreator, config Config) (*Dispatcher, error) {
	if publisher == nil {
		return nil, fmt.Errorf("publisher cannot be nil")
	}
	if watcher == nil {
		return nil, fmt.Errorf("watcher cannot be nil")
	}

	err := mergo.Merge(&config, DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	state := newDispatcherState()
	rec := newRecovery(publisher, watcher, state, config)
	handler := newMessageHandler(messageCreator, publisher, state)

	d := &Dispatcher{
		config:   config,
		watcher:  watcher,
		state:    state,
		recovery: rec,
		stopCh:   make(chan struct{}),
	}

	// Set ourselves as the handler
	if err = watcher.SetHandler(handler); err != nil {
		return nil, fmt.Errorf("failed to set handler: %w", err)
	}

	return d, nil
}

// Start begins processing events and managing async publish results.
// It launches background goroutines for processing publish results,
// checking for stalled messages, and checkpointing progress.
// Returns an error if the dispatcher is already running or if the watcher
// fails to start.
func (d *Dispatcher) Start(ctx context.Context) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return fmt.Errorf("dispatcher already running")
	}
	d.running = true
	d.mu.Unlock()

	d.routinesWg.Add(3) // For the three goroutines

	// Start background processing
	go d.processPublishResults(ctx)
	go d.checkStalledMessages(ctx)
	go d.checkpointLoop(ctx)

	// Start the watcher
	return d.watcher.Start(ctx)
}

// Stop gracefully shuts down the dispatcher and its background goroutines.
func (d *Dispatcher) Stop(ctx context.Context) error {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return nil
	}
	d.running = false
	d.mu.Unlock()

	close(d.stopCh)
	d.watcher.Stop(ctx)

	// Wait with timeout for all goroutines
	done := make(chan struct{})
	go func() {
		d.routinesWg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "dispatcher shutdown timed out")
	case <-done:
		log.Debug().Msg("Dispatcher shutdown completed")
		return nil
	}
}

// processPublishResults continuously processes results from async publishes
func (d *Dispatcher) processPublishResults(ctx context.Context) {
	ticker := time.NewTicker(d.config.ProcessInterval)
	defer ticker.Stop()
	defer d.routinesWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			// Get copy of all pending messages
			msgs := d.state.pending.GetAll()
			log.Debug().Int("numPending", len(msgs)).Msg("Processing pending messages")
			for _, msg := range msgs {
				select {
				case <-msg.future.Done():
					if msg.future.Err() != nil {
						d.recovery.handleError(ctx, msg, msg.future.Err())
					} else {
						d.handlePublishSuccess(ctx, msg)
					}
				default:
					// Future not done yet
				}
			}
		}
	}
}

// handlePublishSuccess processes a successful publish acknowledgment
func (d *Dispatcher) handlePublishSuccess(ctx context.Context, msg *pendingMessage) {
	// log debug message
	log.Ctx(ctx).Debug().EmbedObject(msg).Msg("Message published successfully")

	// Remove all messages up to and including this one since a successful publish
	// with optimistic concurrency guarantees that all previous sequences must
	// have also succeeded.
	d.state.pending.RemoveUpTo(msg.eventSeqNum)

	// Advance lastAckedSeqNum to the highest successful sequence.
	d.state.updateLastAcked(msg.eventSeqNum)

	// Reset failure count on successful publish
	d.recovery.reset()
}

// checkStalledMessages periodically checks for messages that haven't been
// acknowledged within the configured timeout
func (d *Dispatcher) checkStalledMessages(ctx context.Context) {
	ticker := time.NewTicker(d.config.StallCheckInterval)
	defer ticker.Stop()
	defer d.routinesWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			// Get copy of pending messages
			msgs := d.state.pending.GetAll()
			now := time.Now()

			for _, msg := range msgs {
				if now.Sub(msg.publishTime) > d.config.StallTimeout {
					log.Warn().
						Uint64("eventSeq", msg.eventSeqNum).
						Time("publishTime", msg.publishTime).
						Msg("Message publish stalled")
					// Could implement recovery logic here
				}
			}
		}
	}
}

// checkpointLoop periodically saves the last acknowledged sequence number
func (d *Dispatcher) checkpointLoop(ctx context.Context) {
	ticker := time.NewTicker(d.config.CheckpointInterval)
	defer ticker.Stop()
	defer d.routinesWg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			checkpointTarget := d.state.getCheckpointSeqNum()
			// Only checkpoint if we have something new to save
			if checkpointTarget > 0 {
				checkpointCtx, cancel := context.WithTimeout(ctx, d.config.CheckpointTimeout)
				if err := d.watcher.Checkpoint(checkpointCtx, checkpointTarget); err != nil {
					log.Error().Err(err).Msg("Failed to checkpoint watcher")
				} else {
					d.state.updateLastCheckpoint(checkpointTarget)
				}
				cancel()
			}
		}
	}
}
