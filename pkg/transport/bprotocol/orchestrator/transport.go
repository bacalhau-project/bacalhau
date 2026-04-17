// Package orchestrator provides transport layer implementation for orchestrator nodes using
// the legacy bprotocol over NATS. This package will be deprecated in future releases
// in favor of a new transport implementation.
package orchestrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"
	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

// watcherID uniquely identifies the bprotocol dispatcher watcher instance
const watcherID = "orchestrator-bprotocol-dispatcher"

// Config defines the configuration and dependencies required to set up
// the transport layer for an orchestrator node.
type Config struct {
	// NodeID uniquely identifies this orchestrator node in the cluster
	NodeID string

	//NatsConn is the NATS connection to use for communication
	NatsConn *nats.Conn

	// NodeManager handles node discovery, tracking, and health monitoring
	NodeManager nodes.Manager

	// EventStore provides access to the event log for dispatching updates
	EventStore watcher.EventStore

	// ProtocolRouter determines message routing based on node protocol support
	ProtocolRouter *watchers.ProtocolRouter

	// Callback handles responses from compute nodes (bids, results, etc.)
	Callback *orchestrator.Callback
}

// ConnectionManager coordinates all transport-related components for an orchestrator node,
// including:
// - NATS connection management
// - Heartbeat monitoring
// - Node management endpoints
// - Event dispatching to compute nodes
type ConnectionManager struct {
	config              Config
	heartbeatSubscriber ncl.Subscriber  // Receives heartbeats from compute nodes
	dispatcherWatcher   watcher.Watcher // Watches and forwards execution events
}

// NewConnectionManager creates a new ConnectionManager with the given configuration.
// The manager will not be active until Start is called.
func NewConnectionManager(config Config) (*ConnectionManager, error) {
	err := errors.Join(
		validate.NotBlank(config.NodeID, "NodeID cannot be empty"),
		validate.NotNil(config.NatsConn, "NatsConn cannot be nil"),
		validate.NotNil(config.NodeManager, "NodeManager cannot be nil"),
		validate.NotNil(config.EventStore, "EventStore cannot be nil"),
		validate.NotNil(config.ProtocolRouter, "ProtocolRouter cannot be nil"),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid ConnectionManager configuration: %w", err)
	}

	return &ConnectionManager{
		config: config,
	}, nil
}

// Start initializes all transport components in the following order:
// 1. Sets up heartbeat monitoring
// 2. Initializes management endpoints for compute node registration
// 3. Creates compute proxy for job distribution
// 4. Sets up callback handling for compute responses
// 5. Starts event dispatching
//
// If any step fails, all initialized components are cleaned up via Stop.
func (cm *ConnectionManager) Start(ctx context.Context) error {
	var err error
	defer func() {
		if err != nil {
			cm.Stop(ctx)
		}
	}()

	// heartbeat server
	heartbeatServer := NewServer(cm.config.NodeManager)

	// ncl heartbeat subscriber
	cm.heartbeatSubscriber, err = ncl.NewSubscriber(cm.config.NatsConn, ncl.SubscriberConfig{
		Name:            cm.config.NodeID,
		MessageRegistry: bprotocol.MustCreateMessageRegistry(),
		MessageHandler:  heartbeatServer,
	})
	if err != nil {
		return pkgerrors.Wrap(err, "failed to create heartbeat ncl subscriber")
	}
	if err = cm.heartbeatSubscriber.Subscribe(ctx, bprotocol.OrchestratorHeartbeatSubscription()); err != nil {
		return err
	}

	// management handler
	_, err = proxy.NewManagementHandler(proxy.ManagementHandlerParams{
		Conn:               cm.config.NatsConn,
		ManagementEndpoint: heartbeatServer,
	})
	if err != nil {
		return err
	}

	// compute proxy
	computeProxy, err := proxy.NewComputeProxy(proxy.ComputeProxyParams{
		Conn: cm.config.NatsConn,
	})
	if err != nil {
		return err
	}

	// setup callback handler
	_, err = proxy.NewCallbackHandler(proxy.CallbackHandlerParams{
		Name:     cm.config.NodeID,
		Conn:     cm.config.NatsConn,
		Callback: cm.config.Callback,
	})
	if err != nil {
		return err
	}

	// setup bprotocol dispatcher watcher
	cm.dispatcherWatcher, err = watcher.New(ctx, watcherID, cm.config.EventStore,
		watcher.WithHandler(watchers.NewBProtocolDispatcher(watchers.BProtocolDispatcherParams{
			ID:             cm.config.NodeID,
			ComputeService: computeProxy,
			ProtocolRouter: cm.config.ProtocolRouter,
		})),
		watcher.WithEphemeral(),
		watcher.WithAutoStart(),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{jobstore.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithInitialEventIterator(watcher.LatestIterator()))
	if err != nil {
		return fmt.Errorf("failed to setup bprotocol dispatcher watcher: %w", err)
	}

	return nil
}

// Stop gracefully shuts down all transport components in the reverse order
// of initialization. Any errors during shutdown are logged but not returned
// as they cannot be meaningfully handled at this point.
//
// The order of shutdown is important to prevent message loss:
// 1. Stop event dispatcher to prevent new messages to compute nodes
// 2. Stop heartbeat subscriber to prevent false node state updates
func (cm *ConnectionManager) Stop(ctx context.Context) {
	if cm.dispatcherWatcher != nil {
		cm.dispatcherWatcher.Stop(ctx)
		cm.dispatcherWatcher = nil
	}
	if cm.heartbeatSubscriber != nil {
		_ = cm.heartbeatSubscriber.Close(ctx)
		cm.heartbeatSubscriber = nil
	}
}
