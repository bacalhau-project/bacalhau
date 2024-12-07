package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
	"github.com/bacalhau-project/bacalhau/pkg/transport/dispatcher"
)

type DataPlane struct {
	config DataPlaneConfig

	// Core components
	subscriber ncl.Subscriber
	publisher  ncl.OrderedPublisher
	dispatcher *dispatcher.Dispatcher

	// Sequence Trackers
	incomingSequenceTracker *core.SequenceTracker

	// State
	mu      sync.RWMutex
	running bool
}

type DataPlaneConfig struct {
	NodeID    string
	Client    *nats.Conn
	StartFrom uint64 // Sequence number to start dispatching from

	// Message handling
	MessageHandler        ncl.MessageHandler
	MessageCreatorFactory transport.MessageCreatorFactory
	MessageRegistry       *envelope.Registry
	MessageSerializer     envelope.MessageSerializer

	// Event store for event watcher and checkpointing
	StartSeqNum uint64
	EventStore  watcher.EventStore

	// Dispatcher config
	DispatcherConfig dispatcher.Config
}

func (c *DataPlaneConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.NodeID, "nodeID required"),
		validate.NotNil(c.Client, "client required"),
		validate.NotNil(c.EventStore, "event store required"),
	)
}

func NewDataPlane(config DataPlaneConfig) (*DataPlane, error) {
	dp := &DataPlane{
		config:                  config,
		incomingSequenceTracker: core.NewSequenceTracker(),
	}
	return dp, nil
}

func (dp *DataPlane) Start(ctx context.Context) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.running {
		return fmt.Errorf("data plane already running")
	}

	var err error
	defer func() {
		if err != nil {
			if cleanupErr := dp.cleanup(ctx); cleanupErr != nil {
				log.Warn().Err(cleanupErr).Msg("failed to cleanup after start error")
			}
		}
	}()

	incomingSubject := core.NatsSubjectOrchestratorInMsgs(dp.config.NodeID)
	outgoingSubject := core.NatsSubjectOrchestratorOutMsgs(dp.config.NodeID)

	// Create subscriber for compute node messages
	dp.subscriber, err = ncl.NewSubscriber(dp.config.Client, ncl.SubscriberConfig{
		Name:               fmt.Sprintf("orchestrator-%s", dp.config.NodeID),
		MessageRegistry:    dp.config.MessageRegistry,
		MessageSerializer:  dp.config.MessageSerializer,
		MessageHandler:     dp.config.MessageHandler,
		ProcessingNotifier: dp.incomingSequenceTracker,
	})
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Subscribe to incoming messages
	if err = dp.subscriber.Subscribe(ctx, incomingSubject); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Create publisher
	dp.publisher, err = ncl.NewOrderedPublisher(dp.config.Client, ncl.OrderedPublisherConfig{
		Name:              fmt.Sprintf("orchestrator-%s", dp.config.NodeID),
		MessageRegistry:   dp.config.MessageRegistry,
		MessageSerializer: dp.config.MessageSerializer,
		Destination:       outgoingSubject,
	})
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	// Create watcher over event store starting from specified sequence
	var dispatcherWatcher watcher.Watcher
	dispatcherWatcher, err = watcher.New(ctx,
		fmt.Sprintf("orchestrator-dispatcher-%s", dp.config.NodeID),
		dp.config.EventStore,
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithIgnoreCheckpoint(),
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(dp.config.StartSeqNum)),
	)
	if err != nil {
		return fmt.Errorf("failed to create dispatcher watcher: %w", err)
	}

	// disable checkpointing in dispatcher
	dp.config.DispatcherConfig.CheckpointInterval = -1

	// Create dispatcher
	dp.dispatcher, err = dispatcher.New(
		dp.publisher,
		dispatcherWatcher,
		dp.config.MessageCreatorFactory.CreateMessageCreator(ctx, dp.config.NodeID),
		dp.config.DispatcherConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Start dispatcher
	if err = dp.dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	log.Debug().
		Str("nodeID", dp.config.NodeID).
		Str("incoming_subject", incomingSubject).
		Str("outgoing_subject", outgoingSubject).
		Str("start_seq_num", fmt.Sprint(dp.config.StartSeqNum)).
		Str("watcher_id", dispatcherWatcher.ID()).
		Msg("orchestrator to compute data plane started")

	dp.running = true
	return nil
}

func (dp *DataPlane) Stop(ctx context.Context) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if !dp.running {
		return nil
	}

	dp.running = false
	return dp.cleanup(ctx)
}

func (dp *DataPlane) cleanup(ctx context.Context) error {
	var errs error

	if dp.dispatcher != nil {
		if err := dp.dispatcher.Stop(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.dispatcher = nil
	}

	if dp.subscriber != nil {
		if err := dp.subscriber.Close(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.subscriber = nil
	}

	if dp.publisher != nil {
		if err := dp.publisher.Close(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.publisher = nil
	}

	if errs != nil {
		return fmt.Errorf("failed to cleanup data plane: %w", errs)
	}
	return nil
}
