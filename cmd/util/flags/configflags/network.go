package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var NetworkFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   types.OrchestratorEnabledKey,
		DefaultValue: types.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:          "requester",
		ConfigPath:        types.OrchestratorEnabledKey,
		DefaultValue:      types.Default.Orchestrator.Enabled,
		Description:       "When true the orchestrator service will be enabled.",
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorEnabledKey),
	},
	{
		FlagName:          "network-host",
		ConfigPath:        types.OrchestratorHostKey,
		DefaultValue:      types.Default.Orchestrator.Host,
		Description:       `Host to listen for connections from other nodes. Applies to orchestrator nodes.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorHostKey),
	},
	{
		FlagName:          "network-port",
		ConfigPath:        types.OrchestratorPortKey,
		DefaultValue:      types.Default.Orchestrator.Port,
		Description:       `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorPortKey),
	},
	{
		FlagName:          "orchestrators",
		ConfigPath:        types.ComputeOrchestratorsKey,
		DefaultValue:      types.Default.Compute.Orchestrators,
		Description:       `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.ComputeOrchestratorsKey),
	},
	{
		FlagName:          "advertised-address",
		ConfigPath:        types.OrchestratorAdvertiseKey,
		DefaultValue:      types.Default.Orchestrator.Advertise,
		Description:       `Address to advertise to compute nodes to connect to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorAdvertiseKey),
	},
	{
		FlagName:          "cluster-name",
		ConfigPath:        types.OrchestratorClusterNameKey,
		DefaultValue:      types.Default.Orchestrator.Cluster.Name,
		Description:       `Name of the cluster to join.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorClusterNameKey),
	},
	{
		FlagName:          "cluster-host",
		ConfigPath:        types.OrchestratorClusterHostKey,
		DefaultValue:      types.Default.Orchestrator.Cluster.Host,
		Description:       `Address to listen for connections from other orchestrators to form a cluster.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorClusterHostKey),
	},
	{
		FlagName:          "cluster-port",
		ConfigPath:        types.OrchestratorClusterPortKey,
		DefaultValue:      types.Default.Orchestrator.Cluster.Port,
		Description:       `Port to listen for connections from other orchestrators to form a cluster.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorClusterPortKey),
	},
	{
		FlagName:          "cluster-advertised-address",
		ConfigPath:        types.OrchestratorClusterAdvertiseKey,
		DefaultValue:      types.Default.Orchestrator.Cluster.Advertise,
		Description:       `Address to advertise to other orchestrators to connect to.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorClusterAdvertiseKey),
	},
	{
		FlagName:          "cluster-peers",
		ConfigPath:        types.OrchestratorClusterPeersKey,
		DefaultValue:      types.Default.Orchestrator.Cluster.Peers,
		Description:       `Comma-separated list of other orchestrators to connect to to form a cluster.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorClusterPeersKey),
	},
	// deprecated.
	{
		FlagName:          "network-store-dir",
		ConfigPath:        "network.store.deprecated",
		DefaultValue:      "",
		Description:       `Directory that network can use for storage`,
		Deprecated:        true,
		DeprecatedMessage: "network path is no longer configurable, location: $BACALHAU_DIR/orchestrator/nats-store",
	},
}
