package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var ClusterFlags = []Definition{
	{
		FlagName:     "use-nats",
		ConfigPath:   types.NodeClusterUseNATS,
		DefaultValue: Default.Node.Cluster.UseNATS,
		Description:  `Enable NATS transport instead of libp2p.`,
	},
	{
		FlagName:     "cluster-port",
		ConfigPath:   types.NodeClusterPort,
		DefaultValue: Default.Node.Cluster.Port,
		Description:  `Port to listen for connections from other nodes.`,
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   types.NodeClusterOrchestrators,
		DefaultValue: Default.Node.Cluster.Orchestrators,
		Description:  `Comma-separated list of cluster orchestrators to connect to.`,
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   types.NodeClusterAdvertisedAddress,
		DefaultValue: Default.Node.Cluster.AdvertisedAddress,
		Description:  `Address to advertise to other nodes.`,
	},
}
