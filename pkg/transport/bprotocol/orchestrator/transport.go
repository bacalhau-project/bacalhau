// Package orchestrator provides transport layer implementation for orchestrator nodes using
// the legacy bprotocol over NATS. This package will be deprecated in future releases
// in favor of a new transport implementation.
package orchestrator

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	pkgerrors "github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
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

	// ClientFactory creates NATS client connections with the appropriate settings
	ClientFactory natsutil.ClientFactory

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
	natsConn            *nats.Conn      // Connection to NATS server
	nodeManager         nodes.Manager   // Manages compute node lifecycle
	HeartbeatSubscriber ncl.Subscriber  // Receives heartbeats from compute nodes
	DispatcherWatcher   watcher.Watcher // Watches and forwards execution events
}

// NewConnectionManager creates a new ConnectionManager with the given configuration.
// The manager will not be active until Start is called.
func NewConnectionManager(config Config) (*ConnectionManager, error) {
	return &ConnectionManager{
		config: config,
	}, nil
}

// Start initializes all transport components in the following order:
// 1. Establishes NATS connection
// 2. Sets up heartbeat monitoring
// 3. Initializes management endpoints for compute node registration
// 4. Creates compute proxy for job distribution
// 5. Sets up callback handling for compute responses
// 6. Starts event dispatching
//
// If any step fails, all initialized components are cleaned up via Stop.
func (cm *ConnectionManager) Start(ctx context.Context) error {
	var err error
	defer func() {
		if err != nil {
			cm.Stop(ctx)
		}
	}()

	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create NATS client: %s", err)
	}

	// heartbeat server
	heartbeatServer := NewServer(cm.nodeManager)

	// ncl heartbeat subscriber
	cm.HeartbeatSubscriber, err = ncl.NewSubscriber(cm.natsConn, ncl.SubscriberConfig{
		Name:            cm.config.NodeID,
		MessageRegistry: bprotocol.MustCreateMessageRegistry(),
		MessageHandler:  heartbeatServer,
	})
	if err != nil {
		return pkgerrors.Wrap(err, "failed to create heartbeat ncl subscriber")
	}
	if err = cm.HeartbeatSubscriber.Subscribe(ctx, bprotocol.OrchestratorHeartbeatSubscription()); err != nil {
		return err
	}

	// management handler
	_, err = proxy.NewManagementHandler(proxy.ManagementHandlerParams{
		Conn:               cm.natsConn,
		ManagementEndpoint: heartbeatServer,
	})
	if err != nil {
		return err
	}

	// compute proxy
	computeProxy, err := proxy.NewComputeProxy(proxy.ComputeProxyParams{
		Conn: cm.natsConn,
	})
	if err != nil {
		return err
	}

	// setup callback handler
	_, err = proxy.NewCallbackHandler(proxy.CallbackHandlerParams{
		Name:     cm.config.NodeID,
		Conn:     cm.natsConn,
		Callback: cm.config.Callback,
	})
	if err != nil {
		return err
	}

	// setup bprotocol dispatcher watcher
	cm.DispatcherWatcher, err = watcher.New(ctx, watcherID, cm.config.EventStore,
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
// 3. Close NATS connection
func (cm *ConnectionManager) Stop(ctx context.Context) {
	if cm.DispatcherWatcher != nil {
		cm.DispatcherWatcher.Stop(ctx)
		cm.DispatcherWatcher = nil
	}
	if cm.HeartbeatSubscriber != nil {
		cm.HeartbeatSubscriber.Close(ctx)
		cm.HeartbeatSubscriber = nil
	}
	if cm.natsConn != nil {
		cm.natsConn.Close()
		cm.natsConn = nil
	}
}
