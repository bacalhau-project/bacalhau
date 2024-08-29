package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var NetworkFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   types2.OrchestratorEnabledKey,
		DefaultValue: types2.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:     "network-host",
		ConfigPath:   types2.OrchestratorHostKey,
		DefaultValue: types2.Default.Orchestrator.Host,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "network-port",
		ConfigPath:   types2.OrchestratorPortKey,
		DefaultValue: types2.Default.Orchestrator.Port,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   types2.ComputeOrchestratorsKey,
		DefaultValue: types2.Default.Compute.Orchestrators,
		Description:  `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   types2.OrchestratorAdvertiseKey,
		DefaultValue: types2.Default.Orchestrator.Advertise,
		Description:  `Address to advertise to compute nodes to connect to.`,
	},
	{
		FlagName:     "cluster-name",
		ConfigPath:   types2.OrchestratorClusterNameKey,
		DefaultValue: types2.Default.Orchestrator.Cluster.Name,
		Description:  `Name of the cluster to join.`,
	},
	{
		FlagName:     "cluster-host",
		ConfigPath:   types2.OrchestratorClusterHostKey,
		DefaultValue: types2.Default.Orchestrator.Cluster.Host,
		Description:  `Address to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-port",
		ConfigPath:   types2.OrchestratorClusterPortKey,
		DefaultValue: types2.Default.Orchestrator.Cluster.Port,
		Description:  `Port to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-advertised-address",
		ConfigPath:   types2.OrchestratorClusterAdvertiseKey,
		DefaultValue: types2.Default.Orchestrator.Cluster.Advertise,
		Description:  `Address to advertise to other orchestrators to connect to.`,
	},
	{
		FlagName:     "cluster-peers",
		ConfigPath:   types2.OrchestratorClusterPeersKey,
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
