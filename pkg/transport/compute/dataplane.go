package compute

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/core"
	"github.com/bacalhau-project/bacalhau/pkg/transport/dispatcher"
)

const watcherID = "compute-ncl-dispatcher"

type DataPlane struct {
	config DataPlaneConfig

	// Core components
	publisher  ncl.OrderedPublisher
	dispatcher *dispatcher.Dispatcher

	// State
	mu      sync.RWMutex
	running bool
}

type DataPlaneConfig struct {
	NodeID string
	Client *nats.Conn

	// Message handling
	MessageCreator    transport.MessageCreator
	MessageRegistry   *envelope.Registry
	MessageSerializer envelope.MessageSerializer

	// Event store for event watcher
	LastReceivedSeqNum uint64
	EventStore         watcher.EventStore

	// Dispatcher config
	DispatcherConfig dispatcher.Config
	LogStreamServer  logstream.Server
}

func NewDataPlane(config DataPlaneConfig) (*DataPlane, error) {
	dp := &DataPlane{
		config: config,
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

	// create log stream server
	_, err = proxy.NewLogStreamHandler(ctx, proxy.LogStreamHandlerParams{
		Name:            dp.config.NodeID,
		Conn:            dp.config.Client,
		LogstreamServer: dp.config.LogStreamServer,
	})

	// Create publisher
	dp.publisher, err = ncl.NewOrderedPublisher(dp.config.Client, ncl.OrderedPublisherConfig{
		Name:              dp.config.NodeID,
		MessageRegistry:   dp.config.MessageRegistry,
		MessageSerializer: dp.config.MessageSerializer,
		Destination:       core.NatsSubjectComputeOutMsgs(dp.config.NodeID),
	})
	if err != nil {
		return fmt.Errorf("failed to create publisher: %w", err)
	}

	// Create watcher over event store
	var dispatcherWatcher watcher.Watcher
	dispatcherWatcher, err = watcher.New(ctx, watcherID, dp.config.EventStore,
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock),
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(dp.config.LastReceivedSeqNum)))
	if err != nil {
		return fmt.Errorf("failed to create dispatcher watcher: %w", err)
	}

	// Create dispatcher
	dp.dispatcher, err = dispatcher.New(
		dp.publisher,
		dispatcherWatcher,
		dp.config.MessageCreator,
		dp.config.DispatcherConfig,
	)
	if err != nil {
		return fmt.Errorf("failed to create dispatcher: %w", err)
	}

	// Start dispatcher
	if err = dp.dispatcher.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

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
		if err := dp.dispatcher.Stop(context.Background()); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.dispatcher = nil
	}

	if dp.publisher != nil {
		if err := dp.publisher.Close(context.Background()); err != nil {
			errs = errors.Join(errs, err)
		}
		dp.publisher = nil
	}

	if errs != nil {
		return fmt.Errorf("failed to cleanup data plane: %w", errs)
	}
	return nil
}
