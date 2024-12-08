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

const watcherID = "orchestrator-bprotocol-dispatcher"

type Config struct {
	NodeID string

	ClientFactory natsutil.ClientFactory

	NodeManager    nodes.Manager
	EventStore     watcher.EventStore
	ProtocolRouter *watchers.ProtocolRouter
	Callback       *orchestrator.Callback
}

type ConnectionManager struct {
	config Config

	natsConn            *nats.Conn
	nodeManager         nodes.Manager
	HeartbeatSubscriber ncl.Subscriber
	DispatcherWatcher   watcher.Watcher
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

// Stop stops the connection manager
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
