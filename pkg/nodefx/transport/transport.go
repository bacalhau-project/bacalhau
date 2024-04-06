package transport

import (
	"context"
	"fmt"

	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configfx"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
)

var Module = fx.Module("transport",
	fx.Provide(LoadConfig),
	fx.Provide(NATS),
)

func LoadConfig(nodeID types.NodeID, nodeKind types.NodeKind, c *configfx.Config) (nats_transport.NATSTransportConfig, error) {
	var networkConfig types.NetworkConfig
	if err := c.ForKey(types.NodeNetwork, &networkConfig); err != nil {
		return nats_transport.NATSTransportConfig{}, err
	}
	var clusterConfig types.NetworkClusterConfig
	if err := c.ForKey(types.NodeNetworkCluster, &clusterConfig); err != nil {
		return nats_transport.NATSTransportConfig{}, err
	}
	return nats_transport.NATSTransportConfig{
		NodeID:                   string(nodeID),
		IsRequesterNode:          nodeKind.IsRequester,
		Port:                     networkConfig.Port,
		AdvertisedAddress:        networkConfig.AdvertisedAddress,
		AuthSecret:               networkConfig.AuthSecret,
		Orchestrators:            networkConfig.Orchestrators,
		StoreDir:                 networkConfig.StoreDir,
		ClusterName:              clusterConfig.Name,
		ClusterPort:              clusterConfig.Port,
		ClusterPeers:             clusterConfig.Peers,
		ClusterAdvertisedAddress: clusterConfig.AdvertisedAddress,
	}, nil
}

func NATS(cfg nats_transport.NATSTransportConfig) (*nats_transport.NATSTransport, error) {
	// TODO(forrest): NewNATSTransport needs to be broken up into the parts that
	// create the transport and the parts that connect, i.e. the constructor does too much.
	// once that is complete use a lifecycle here to start and stop the transport.
	natsTransportLayer, err := nats_transport.NewNATSTransport(context.TODO(), cfg)
	if err != nil {
		return nil, fmt.Errorf("creating NATS transport layer: %w", err)
	}
	return natsTransportLayer, nil
}
