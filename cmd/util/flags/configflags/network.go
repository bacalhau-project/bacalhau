package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var NetworkFlags = []Definition{
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
}
