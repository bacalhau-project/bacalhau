package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var NetworkFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   "Orchestrator.Enabled",
		DefaultValue: types2.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:     "network-host",
		ConfigPath:   "Orchestrator.Host",
		DefaultValue: types2.Default.Orchestrator.Host,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "network-port",
		ConfigPath:   "Orchestrator.Port",
		DefaultValue: types2.Default.Orchestrator.Port,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   "Compute.Orchestrators",
		DefaultValue: types2.Default.Compute.Orchestrators,
		Description:  `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   "Orchestrator.Advertise",
		DefaultValue: types2.Default.Orchestrator.Advertise,
		Description:  `Address to advertise to compute nodes to connect to.`,
	},
	{
		FlagName:     "cluster-name",
		ConfigPath:   "Orchestrator.Cluster.Name",
		DefaultValue: types2.Default.Orchestrator.Cluster.Name,
		Description:  `Name of the cluster to join.`,
	},
	{
		FlagName:     "cluster-host",
		ConfigPath:   "Orchestrator.Cluster.Host",
		DefaultValue: types2.Default.Orchestrator.Cluster.Host,
		Description:  `Address to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-port",
		ConfigPath:   "Orchestrator.Cluster.Port",
		DefaultValue: types2.Default.Orchestrator.Cluster.Port,
		Description:  `Port to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-advertised-address",
		ConfigPath:   "Orchestrator.Cluster.Advertise",
		DefaultValue: types2.Default.Orchestrator.Cluster.Advertise,
		Description:  `Address to advertise to other orchestrators to connect to.`,
	},
	{
		FlagName:     "cluster-peers",
		ConfigPath:   "Orchestrator.Cluster.Peers",
		DefaultValue: types2.Default.Orchestrator.Cluster.Peers,
		Description:  `Comma-separated list of other orchestrators to connect to to form a cluster.`,
	},
	// deprecated.
	{
		FlagName:          "network-store-dir",
		ConfigPath:        "network.store.deprecated",
		DefaultValue:      "",
		Description:       `Directory that network can use for storage`,
		FailIfUsed:        true,
		Deprecated:        true,
		DeprecatedMessage: "network path is no longer configurable, location: $BACALHAU_DIR/orchestrator/nats-store",
	},
}
