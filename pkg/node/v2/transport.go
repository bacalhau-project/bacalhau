package v2

import (
	"context"
	"fmt"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func SetupTransport(name string, r *repo.FsRepo, cfg v2.Bacalhau) (*nats_transport.NATSTransport, error) {
	networkStorePath, err := r.NetworkTransportStorePath()
	if err != nil {
		return nil, fmt.Errorf("reading network transport path: %w", err)
	}

	// TODO the context passed here is only used for logging
	transport, err := nats_transport.NewNATSTransport(context.TODO(),
		&nats_transport.NATSTransportConfig{
			NodeID:                   name,
			StoreDir:                 networkStorePath,
			Port:                     cfg.Orchestrator.Port,
			AdvertisedAddress:        cfg.Orchestrator.Advertise,
			IsRequesterNode:          cfg.Orchestrator.Enabled,
			AuthSecret:               cfg.Orchestrator.AuthSecret,
			ClusterName:              cfg.Orchestrator.Cluster.Name,
			ClusterPort:              cfg.Orchestrator.Cluster.Port,
			ClusterAdvertisedAddress: cfg.Orchestrator.Cluster.Advertise,
			ClusterPeers:             cfg.Orchestrator.Cluster.Peers,

			// TODO we need different setup code for client vs server, Orchestrators is usually a client value
			Orchestrators: cfg.Compute.Orchestrators,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create NATS transport: %w", err)
	}
	return transport, nil
}
