package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var NetworkFlags = []Definition{
	{
		FlagName:          "network",
		ConfigPath:        types.NodeNetworkType,
		DefaultValue:      Default.Node.Network.Type,
		Description:       `Inter-node network layer type (e.g. nats, libp2p).`,
		Deprecated:        true,
		DeprecatedMessage: "The libp2p transport will be deprecated in a future version in favor of NATS",
	},
	{
		FlagName:     "network-port",
		ConfigPath:   types.NodeNetworkPort,
		DefaultValue: Default.Node.Network.Port,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "network-store-dir",
		ConfigPath:   types.NodeNetworkStoreDir,
		DefaultValue: Default.Node.Network.StoreDir,
		Description:  `Directory that network can use for storage`,
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   types.NodeNetworkOrchestrators,
		DefaultValue: Default.Node.Network.Orchestrators,
		Description:  `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   types.NodeNetworkAdvertisedAddress,
		DefaultValue: Default.Node.Network.AdvertisedAddress,
		Description:  `Address to advertise to compute nodes to connect to.`,
	},
	{
		FlagName:     "cluster-name",
		ConfigPath:   types.NodeNetworkClusterName,
		DefaultValue: Default.Node.Network.Cluster.Name,
		Description:  `Name of the cluster to join.`,
	},
	{
		FlagName:     "cluster-port",
		ConfigPath:   types.NodeNetworkClusterPort,
		DefaultValue: Default.Node.Network.Cluster.Port,
		Description:  `Port to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-advertised-address",
		ConfigPath:   types.NodeNetworkClusterAdvertisedAddress,
		DefaultValue: Default.Node.Network.Cluster.AdvertisedAddress,
		Description:  `Address to advertise to other orchestrators to connect to.`,
	},
	{
		FlagName:     "cluster-peers",
		ConfigPath:   types.NodeNetworkClusterPeers,
		DefaultValue: Default.Node.Network.Cluster.Peers,
		Description:  `Comma-separated list of other orchestrators to connect to to form a cluster.`,
	},
}
