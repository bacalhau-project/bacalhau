package compute

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol"
	"github.com/bacalhau-project/bacalhau/pkg/transport/nclprotocol/dispatcher"
)

// watcherID is the unique identifier for the data plane event watcher
const watcherID = "compute-ncl-dispatcher"

// DataPlane manages the data transfer operations between a compute node and the orchestrator.
// It is responsible for:
// - Setting up and managing the log streaming server
// - Reliable message publishing through ordered publisher
// - Event watching and dispatching
// - Maintaining message sequence ordering
type DataPlane struct {
	config Config // Global configuration

	// Core messaging components
	Client     *nats.Conn             // NATS connection for messaging
	publisher  ncl.OrderedPublisher   // Handles ordered message publishing
	dispatcher *dispatcher.Dispatcher // Manages event watching and dispatch

	// Sequence tracking
	lastReceivedSeqNum uint64 // Last sequence number received from orchestrator

	// State management
	mu      sync.RWMutex // Protects state changes
	running bool         // Indicates if data plane is active
}

// DataPlaneParams encapsulates the parameters needed to create a new DataPlane
type DataPlaneParams struct {
	Config             Config
	Client             *nats.Conn // NATS client connection
	LastReceivedSeqNum uint64     // Initial sequence number for message ordering
}

// NewDataPlane creates a new DataPlane instance with the provided parameters.
// It initializes the data plane but does not start any operations - Start() must be called.
func NewDataPlane(params DataPlaneParams) (*DataPlane, error) {
	dp := &DataPlane{
		config:             params.Config,
		Client:             params.Client,
		lastReceivedSeqNum: params.LastReceivedSeqNum,
	}
	return dp, nil
}

// Start initializes and begins data plane operations. This includes:
// 1. Setting up the log stream server for job output streaming
// 2. Creating an ordered publisher for reliable message delivery
// 3. Setting up event watching and dispatching
// 4. Starting the dispatcher
//
// Note that message subscriber and handler are not started here, as they must be started
// during the handshake and before the data plane is started to avoid message loss.
//
// If any component fails to initialize, cleanup is performed before returning error.
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

	// Set up log streaming for job output
	_, err = proxy.NewLogStreamHandler(ctx, proxy.LogStreamHandlerParams{
		Name:            dp.config.NodeID,
		Conn:            dp.Client,
		LogstreamServer: dp.config.LogStreamServer,
	})
	if err != nil {
		return fmt.Errorf("failed to set up log stream handler: %w", err)
	}
	// Initialize ordered publisher for reliable message delivery
	dp.publisher, err = ncl.NewOrderedPublisher(dp.Client, ncl.OrderedPublisherConfig{
		Name:              dp.config.NodeID,
		MessageRegistry:   dp.config.MessageRegistry,
		MessageSerializer: dp.config.MessageSerializer,
		Destination:       nclprotocol.NatsSubjectComputeOutMsgs(dp.config.NodeID),
	})
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	// Create event watcher starting from last known sequence
	var dispatcherWatcher watcher.Watcher
	dispatcherWatcher, err = watcher.New(ctx, watcherID, dp.config.EventStore,
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock),
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(dp.lastReceivedSeqNum)),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create dispatcher watcher: %w", err)
	}

	// Initialize dispatcher to handle event watching and publishing
	dp.dispatcher, err = dispatcher.New(
		dp.publisher,
		dispatcherWatcher,
		dp.config.DataPlaneMessageCreator,
		dp.config.DispatcherConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Start the dispatcher
	if err = dp.dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	dp.running = true
	return nil
}

// Stop gracefully shuts down all data plane operations.
// It ensures proper cleanup of resources by:
// 1. Stopping the dispatcher
// 2. Closing the publisher
// Any errors during cleanup are collected and returned.
func (dp *DataPlane) Stop(ctx context.Context) error {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	if !dp.running {
		return nil
	}

	dp.running = false
	return dp.cleanup(ctx)
}

// cleanup handles the orderly shutdown of data plane components.
// It ensures resources are released in the correct order and collects any errors.
func (dp *DataPlane) cleanup(ctx context.Context) error {
	var errs error

	// Stop dispatcher first to prevent new messages
	if dp.dispatcher != nil {
		if err := dp.dispatcher.Stop(ctx); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.dispatcher = nil
	}

	// Then close the publisher
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
