package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var NetworkFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   cfgtypes.OrchestratorEnabledKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:     "network-host",
		ConfigPath:   cfgtypes.OrchestratorHostKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Host,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "network-port",
		ConfigPath:   cfgtypes.OrchestratorPortKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Port,
		Description:  `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   cfgtypes.ComputeOrchestratorsKey,
		DefaultValue: cfgtypes.Default.Compute.Orchestrators,
		Description:  `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   cfgtypes.OrchestratorAdvertiseKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Advertise,
		Description:  `Address to advertise to compute nodes to connect to.`,
	},
	{
		FlagName:     "cluster-name",
		ConfigPath:   cfgtypes.OrchestratorClusterNameKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Cluster.Name,
		Description:  `Name of the cluster to join.`,
	},
	{
		FlagName:     "cluster-host",
		ConfigPath:   cfgtypes.OrchestratorClusterHostKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Cluster.Host,
		Description:  `Address to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-port",
		ConfigPath:   cfgtypes.OrchestratorClusterPortKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Cluster.Port,
		Description:  `Port to listen for connections from other orchestrators to form a cluster.`,
	},
	{
		FlagName:     "cluster-advertised-address",
		ConfigPath:   cfgtypes.OrchestratorClusterAdvertiseKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Cluster.Advertise,
		Description:  `Address to advertise to other orchestrators to connect to.`,
	},
	{
		FlagName:     "cluster-peers",
		ConfigPath:   cfgtypes.OrchestratorClusterPeersKey,
		DefaultValue: cfgtypes.Default.Orchestrator.Cluster.Peers,
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
