package v2

import (
	"context"
	"fmt"

	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
)

func SetupTransport(cfg v2.Bacalhau) (*nats_transport.NATSTransport, error) {
	transport, err := nats_transport.NewNATSTransport(context.TODO(),
		&nats_transport.NATSTransportConfig{
			NodeID:                   cfg.Name,
			Port:                     "TODO",
			AdvertisedAddress:        "TODO",
			Orchestrators:            "TODO",
			IsRequesterNode:          cfg.Orchestrator.Enabled,
			StoreDir:                 "TODO",
			AuthSecret:               "TODO",
			ClusterName:              "TODO",
			ClusterPort:              "TODO",
			ClusterAdvertisedAddress: cfg.Orchestrator.Cluster.Advertise,
			ClusterPeers:             cfg.Orchestrator.Cluster.Peers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create NATS transport: %w", err)
	}
	return transport, nil
}
