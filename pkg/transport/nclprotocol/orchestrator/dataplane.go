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
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
)

// DataPlane manages the message flow between orchestrator and a single compute node.
// It handles:
// - Reliable message delivery through ordered publisher
// - Sequence tracking for both incoming and outgoing messages
// - Event watching and dispatching
// Each DataPlane instance corresponds to one compute node connection.
type DataPlane struct {
	config DataPlaneConfig

	// Core messaging components
	subscriber ncl.Subscriber         // Handles incoming messages from compute node
	publisher  ncl.OrderedPublisher   // Sends messages to compute node
	dispatcher *dispatcher.Dispatcher // Manages event watching and dispatch

	// Sequence Trackers
	incomingSequenceTracker *nclprotocol.SequenceTracker

	// State management
	mu      sync.RWMutex // Protects state changes
	running bool         // Indicates if data plane is active
}

// DataPlaneConfig defines the configuration for a DataPlane instance.
// Each config corresponds to a single compute node connection.
type DataPlaneConfig struct {
	NodeID string     // ID of the compute node this data plane serves
	Client *nats.Conn // NATS connection

	// Message handling
	MessageHandler        ncl.MessageHandler
	MessageCreatorFactory nclprotocol.MessageCreatorFactory
	MessageRegistry       *envelope.Registry
	MessageSerializer     envelope.MessageSerializer

	// Event tracking
	EventStore  watcher.EventStore
	StartSeqNum uint64 // Initial sequence for event watching

	// Dispatcher settings
	DispatcherConfig dispatcher.Config
}

func (c *DataPlaneConfig) Validate() error {
	return errors.Join(
		validate.NotBlank(c.NodeID, "nodeID required"),
		validate.NotNil(c.Client, "client required"),
		validate.NotNil(c.EventStore, "event store required"),
		validate.NotNil(c.MessageHandler, "message handler required"),
		validate.NotNil(c.MessageCreatorFactory, "message creator factory required"),
		validate.NotNil(c.MessageRegistry, "message registry required"),
		validate.NotNil(c.MessageSerializer, "message serializer required"),
	)
}

// NewDataPlane creates a new DataPlane instance for a compute node.
func NewDataPlane(config DataPlaneConfig) (*DataPlane, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &DataPlane{
		config:                  config,
		incomingSequenceTracker: nclprotocol.NewSequenceTracker(),
	}, nil
}

// Start initializes and begins data plane operations. This includes:
// 1. Creating subscriber for compute node messages
// 2. Creating ordered publisher for reliable delivery
// 3. Setting up event watching and dispatching
// 4. Starting all components in correct order
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

	// Define NATS subjects for this compute node
	inSubject := nclprotocol.NatsSubjectOrchestratorInMsgs(dp.config.NodeID)
	outSubject := nclprotocol.NatsSubjectOrchestratorOutMsgs(dp.config.NodeID)

	// Set up subscriber for incoming messages
	if err = dp.setupSubscriber(ctx, inSubject); err != nil {
		return fmt.Errorf("failed to setup subscriber: %w", err)
	}

	// Set up publisher for outgoing messages
	if err = dp.setupPublisher(outSubject); err != nil {
		return fmt.Errorf("failed to setup publisher: %w", err)
	}

	// Set up dispatcher for event watching
	if err = dp.setupDispatcher(ctx); err != nil {
		return fmt.Errorf("failed to setup dispatcher: %w", err)
	}

	log.Debug().
		Str("nodeID", dp.config.NodeID).
		Str("incomingSubject", inSubject).
		Str("outgoingSubject", outSubject).
		Uint64("startSeqNum", dp.config.StartSeqNum).
		Msg("Data plane started")

	dp.running = true
	return nil
}

func (dp *DataPlane) setupSubscriber(ctx context.Context, subject string) error {
	var err error
	dp.subscriber, err = ncl.NewSubscriber(dp.config.Client, ncl.SubscriberConfig{
		Name:               fmt.Sprintf("orchestrator-%s", dp.config.NodeID),
		MessageRegistry:    dp.config.MessageRegistry,
		MessageSerializer:  dp.config.MessageSerializer,
		MessageHandler:     dp.config.MessageHandler,
		ProcessingNotifier: dp.incomingSequenceTracker,
	})
	if err != nil {
		return fmt.Errorf("create subscriber: %w", err)
	}

	return dp.subscriber.Subscribe(ctx, subject)
}

func (dp *DataPlane) setupPublisher(subject string) error {
	var err error
	dp.publisher, err = ncl.NewOrderedPublisher(dp.config.Client, ncl.OrderedPublisherConfig{
		Name:              fmt.Sprintf("orchestrator-%s", dp.config.NodeID),
		MessageRegistry:   dp.config.MessageRegistry,
		MessageSerializer: dp.config.MessageSerializer,
		Destination:       subject,
	})
	if err != nil {
		return fmt.Errorf("create publisher: %w", err)
	}
	return nil
}

func (dp *DataPlane) setupDispatcher(ctx context.Context) error {
	// Create watcher starting from specified sequence
	dispatcherWatcher, err := watcher.New(ctx,
		fmt.Sprintf("orchestrator-dispatcher-%s", dp.config.NodeID),
		dp.config.EventStore,
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock),
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(dp.config.StartSeqNum)),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
	)
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}

	// Create message creator for this compute node
	messageCreator, err := dp.config.MessageCreatorFactory.CreateMessageCreator(
		ctx, dp.config.NodeID)
	if err != nil {
		return fmt.Errorf("create message creator: %w", err)
	}

	// Disable checkpointing in dispatcher since we handle it elsewhere
	config := dp.config.DispatcherConfig
	config.CheckpointInterval = -1

	// Create and start dispatcher
	dp.dispatcher, err = dispatcher.New(
		dp.publisher,
		dispatcherWatcher,
		messageCreator,
		config,
	)
	if err != nil {
		return fmt.Errorf("create dispatcher: %w", err)
	}

	if err = dp.dispatcher.Start(context.TODO()); err != nil {
		return fmt.Errorf("start dispatcher: %w", err)
	}

	return nil
}

// Stop gracefully shuts down all data plane operations.
// It ensures proper cleanup of resources by stopping components
// in correct order: dispatcher -> subscriber -> publisher
func (dp *DataPlane) Stop(ctx context.Context) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if !dp.running {
		return nil
	}

	dp.running = false
	return dp.cleanup(ctx)
}

// cleanup handles orderly shutdown of data plane components
func (dp *DataPlane) cleanup(ctx context.Context) error {
	var errs error

	// Stop dispatcher first to prevent new messages
	if dp.dispatcher != nil {
		if err := dp.dispatcher.Stop(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("stop dispatcher: %w", err))
		}
		dp.dispatcher = nil
	}

	// Then clean up subscriber
	if dp.subscriber != nil {
		if err := dp.subscriber.Close(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("close subscriber: %w", err))
		}
		dp.subscriber = nil
	}

	// Finally clean up publisher
	if dp.publisher != nil {
		if err := dp.publisher.Close(ctx); err != nil {
			errs = errors.Join(errs, fmt.Errorf("close publisher: %w", err))
		}
		dp.publisher = nil
	}

	if errs != nil {
		return fmt.Errorf("cleanup failed: %w", errs)
	}
	return nil
}

// GetLastProcessedSequence returns the last sequence number processed
// from incoming messages from this compute node
func (dp *DataPlane) GetLastProcessedSequence() uint64 {
	return dp.incomingSequenceTracker.GetLastSeqNum()
}
