package orchestrator

import (
	"context"
	"time"

	pkgerrors "github.com/pkg/errors"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/kvstore"
	"github.com/bacalhau-project/bacalhau/pkg/routing/tracing"
)

const (
	HeartbeatTopic = "heartbeat"
)

func SetupNodeManager(
	ctx context.Context,
	transportLayer *nats_transport.NATSTransport,
	nodeInfoStore routing.NodeInfoStore,
	cfg v2.Orchestrator,
) (*manager.NodeManager, error) {
	// heartbeat service
	heartbeatParams := heartbeat.HeartbeatServerParams{
		Client: transportLayer.Client(),
		Topic:  HeartbeatTopic,
		// NB(forrest): this was pulled from the default config
		CheckFrequency:        time.Second * 30,
		NodeDisconnectedAfter: time.Duration(cfg.NodeManager.DisconnectTimeout),
	}
	heartbeatSvr, err := heartbeat.NewServer(heartbeatParams)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create heartbeat server using NATS transport connection info")
	}

	defaultMembership := models.NodeMembership.PENDING
	if cfg.NodeManager.AutoApprove {
		defaultMembership = models.NodeMembership.APPROVED
	}

	// node manager
	// Create a new node manager to keep track of compute nodes connecting
	// to the network. Provide it with a mechanism to lookup (and enhance)
	// node info, and a reference to the heartbeat server
	return manager.NewNodeManager(manager.NodeManagerParams{
		NodeInfo:             nodeInfoStore,
		Heartbeats:           heartbeatSvr,
		DefaultApprovalState: defaultMembership,
	}), nil
}

func SetupNodeInfoStore(ctx context.Context, transportLayer *nats_transport.NATSTransport) (routing.NodeInfoStore, error) {
	// nodeInfoStore
	nodeInfoStore, err := kvstore.NewNodeStore(ctx, kvstore.NodeStoreParams{
		BucketName: kvstore.BucketNameCurrent,
		Client:     transportLayer.Client(),
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create node info store using NATS transport connection info")
	}

	tracingInfoStore := tracing.NewNodeStore(nodeInfoStore)

	return tracingInfoStore, nil
}
