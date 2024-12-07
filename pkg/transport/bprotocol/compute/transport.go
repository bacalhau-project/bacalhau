package compute

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/watchers"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	natsutil "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/nats/proxy"
	"github.com/bacalhau-project/bacalhau/pkg/transport/bprotocol"
)

const watcherID = "compute-bprotocol-dispatcher"

type Config struct {
	NodeID     string
	ComputeDir string

	ClientFactory natsutil.ClientFactory

	NodeInfoProvider models.NodeInfoProvider
	HeartbeatConfig  types.Heartbeat
	ComputeEndpoint  compute.Endpoint
	EventStore       watcher.EventStore
}

type ConnectionManager struct {
	config Config

	natsConn           *nats.Conn
	heartbeatPublisher ncl.Publisher
	heartbeatClient    *HeartbeatClient
	managementClient   *ManagementClient
	dispatcherWatcher  watcher.Watcher
}

func NewConnectionManager(config Config) (*ConnectionManager, error) {
	return &ConnectionManager{
		config: config,
	}, nil
}

// Start starts the connection manager
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

	_, err = proxy.NewComputeHandler(ctx, proxy.ComputeHandlerParams{
		Name:            cm.config.NodeID,
		Conn:            cm.natsConn,
		ComputeEndpoint: cm.config.ComputeEndpoint,
	})
	if err != nil {
		return err
	}

	regFilename := fmt.Sprintf("%s.registration.lock", cm.config.NodeID)
	regFilename = filepath.Join(cm.config.ComputeDir, regFilename)

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
		RegistrationFilePath: regFilename,
		HeartbeatClient:      cm.heartbeatClient,
		HeartbeatConfig:      cm.config.HeartbeatConfig,
	})
	if err = managementClient.RegisterNode(ctx); err != nil {
		if strings.Contains(err.Error(), bprotocol.ErrUpgradeAvailable.Error()) {
			log.Info().Msg("Disabling bprotocol management client due to upgrade available")
			cm.Stop(ctx)
			return nil
		}
		return fmt.Errorf("failed to register node with requester: %s", err)
	}

	// Start the management client
	go managementClient.Start(ctx)

	// setup bprotocol dispatcher watcher
	cm.dispatcherWatcher, err = watcher.New(ctx, watcherID, cm.config.EventStore,
		watcher.WithHandler(watchers.NewBProtocolDispatcher(proxy.NewCallbackProxy(proxy.CallbackProxyParams{
			Conn: cm.natsConn,
		}))),
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

// Stop stops the connection manager
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
		cm.heartbeatClient.Close(ctx)
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
