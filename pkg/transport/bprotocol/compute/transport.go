// Package compute provides transport layer implementation for compute nodes using
// the legacy bprotocol over NATS. This package will be deprecated in future releases
// in favor of a new transport implementation.
package compute

import (
	"context"
	"errors"
	"fmt"

	"github.com/nats-io/nats.go"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

// watcherID uniquely identifies the bprotocol dispatcher watcher instance
const watcherID = "compute-bprotocol-dispatcher"

// Config defines the configuration and dependencies required to set up
// the transport layer for a compute node.
type Config struct {
	// NodeID uniquely identifies this compute node in the cluster
	NodeID string

	// ClientFactory creates NATS client connections with the appropriate settings
	ClientFactory natsutil.ClientFactory

	// NodeInfoProvider supplies current node information for registration and updates
	NodeInfoProvider models.NodeInfoProvider

	// HeartbeatConfig controls heartbeat timing and behavior
	HeartbeatConfig types.Heartbeat

	// ComputeEndpoint handles incoming compute requests
	ComputeEndpoint compute.Endpoint

	// EventStore provides access to the event log for dispatching updates
	EventStore watcher.EventStore
}

// ConnectionManager coordinates all transport-related components for a compute node,
// including:
// - NATS connection management
// - Heartbeat publishing
// - Node registration and updates
// - Event dispatching to orchestrator
type ConnectionManager struct {
	config             Config
	natsConn           *nats.Conn
	heartbeatPublisher ncl.Publisher     // Publishes heartbeats to orchestrator
	heartbeatClient    *HeartbeatClient  // Manages heartbeat timing and sequencing
	managementClient   *ManagementClient // Handles node registration and updates
	dispatcherWatcher  watcher.Watcher   // Watches and forwards execution events
}

// NewConnectionManager creates a new ConnectionManager with the given configuration.
// The manager will not be active until Start is called.
func NewConnectionManager(config Config) (*ConnectionManager, error) {
	err := errors.Join(
		validate.NotNil(config.NodeID, "NodeID cannot be empty"),
		validate.NotNil(config.ClientFactory, "ClientFactory cannot be nil"),
		validate.NotNil(config.NodeInfoProvider, "NodeInfoProvider cannot be nil"),
		validate.NotNil(config.ComputeEndpoint, "ComputeEndpoint cannot be nil"),
		validate.NotNil(config.EventStore, "EventStore cannot be nil"),
		validate.NotNil(config.HeartbeatConfig, "HeartbeatConfig cannot be nil"),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid ConnectionManager configuration: %w", err)
	}
	return &ConnectionManager{
		config: config,
	}, nil
}

// Start initializes all transport components in the following order:
// 1. Establishes NATS connection
// 2. Sets up compute request handling
// 3. Initializes heartbeat publishing
// 4. Registers with orchestrator
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

	cm.natsConn, err = cm.config.ClientFactory.CreateClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create NATS client: %s", err)
	}

	// register nats compute handler
	_, err = proxy.NewComputeHandler(ctx, proxy.ComputeHandlerParams{
		Name:            cm.config.NodeID,
		Conn:            cm.natsConn,
		ComputeEndpoint: cm.config.ComputeEndpoint,
	})
	if err != nil {
		return err
	}

	// create nats callback proxy
	callbackProxy := proxy.NewCallbackProxy(proxy.CallbackProxyParams{
		Conn: cm.natsConn,
	})

	// heartbeat client
	cm.heartbeatPublisher, err = ncl.NewPublisher(cm.natsConn, ncl.PublisherConfig{
		Name:            cm.config.NodeID,
		Destination:     bprotocol.ComputeHeartbeatTopic(cm.config.NodeID),
		MessageRegistry: bprotocol.MustCreateMessageRegistry(),
	})
	if err != nil {
		return err
	}

	cm.heartbeatClient, err = NewHeartbeatClient(cm.config.NodeID, cm.heartbeatPublisher)
	if err != nil {
		return err
	}

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	managementClient := NewManagementClient(&ManagementClientParams{
		NodeInfoProvider: cm.config.NodeInfoProvider,
		ManagementProxy: proxy.NewManagementProxy(proxy.ManagementProxyParams{
			Conn: cm.natsConn,
		}),
		HeartbeatClient: cm.heartbeatClient,
		HeartbeatConfig: cm.config.HeartbeatConfig,
	})
	if err = managementClient.RegisterNode(ctx); err != nil {
		return err
	}

	// Start the management client
	go managementClient.Start(ctx)

	// setup bprotocol dispatcher watcher
	cm.dispatcherWatcher, err = watcher.New(ctx, watcherID, cm.config.EventStore,
		watcher.WithHandler(watchers.NewBProtocolDispatcher(callbackProxy)),
		watcher.WithEphemeral(),
		watcher.WithAutoStart(),
		watcher.WithFilter(watcher.EventFilter{
			ObjectTypes: []string{compute.EventObjectExecutionUpsert},
		}),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip),
		watcher.WithMaxRetries(3),
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
// 1. Stop event dispatcher to prevent new messages
// 2. Stop management client to prevent new registrations
// 3. Stop heartbeat client to prevent false positive disconnections
// 4. Close NATS connection
func (cm *ConnectionManager) Stop(ctx context.Context) {
	if cm.dispatcherWatcher != nil {
		cm.dispatcherWatcher.Stop(ctx)
		cm.dispatcherWatcher = nil
	}
	if cm.managementClient != nil {
		cm.managementClient.Stop()
		cm.managementClient = nil
	}
	if cm.heartbeatClient != nil {
		_ = cm.heartbeatClient.Close(ctx)
		cm.heartbeatClient = nil
	}
	if cm.heartbeatPublisher != nil {
		cm.heartbeatPublisher = nil
	}
	if cm.natsConn != nil {
		cm.natsConn.Close()
		cm.natsConn = nil
	}
}
