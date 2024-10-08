package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var NetworkFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   types.OrchestratorEnabledKey,
		DefaultValue: config.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:          "requester",
		ConfigPath:        types.OrchestratorEnabledKey,
		DefaultValue:      config.Default.Orchestrator.Enabled,
		Description:       "When true the orchestrator service will be enabled.",
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorEnabledKey),
	},
	{
		FlagName:          "network-host",
		ConfigPath:        types.OrchestratorHostKey,
		DefaultValue:      config.Default.Orchestrator.Host,
		Description:       `Host to listen for connections from other nodes. Applies to orchestrator nodes.`,
		Deprecated:        true,
		DeprecatedMessage: makeDeprecationMessage(types.OrchestratorHostKey),
	},
	{
		FlagName:             "network-port",
		ConfigPath:           types.OrchestratorPortKey,
		DefaultValue:         config.Default.Orchestrator.Port,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_PORT"},
		Description:          `Port to listen for connections from other nodes. Applies to orchestrator nodes.`,
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorPortKey),
	},
	{
		FlagName:             "orchestrators",
		ConfigPath:           types.ComputeOrchestratorsKey,
		DefaultValue:         config.Default.Compute.Orchestrators,
		Description:          `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_ORCHESTRATORS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.ComputeOrchestratorsKey),
	},
	{
		FlagName:             "advertised-address",
		ConfigPath:           types.OrchestratorAdvertiseKey,
		DefaultValue:         config.Default.Orchestrator.Advertise,
		Description:          `Address to advertise to compute nodes to connect to.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_ADVERTISEDADDRESS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorAdvertiseKey),
	},
	{
		FlagName:             "cluster-name",
		ConfigPath:           types.OrchestratorClusterNameKey,
		DefaultValue:         config.Default.Orchestrator.Cluster.Name,
		Description:          `Name of the cluster to join.`,
		Deprecated:           true,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_CLUSTER_NAME"},
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorClusterNameKey),
	},
	{
		FlagName:             "cluster-host",
		ConfigPath:           types.OrchestratorClusterHostKey,
		DefaultValue:         config.Default.Orchestrator.Cluster.Host,
		Description:          `Address to listen for connections from other orchestrators to form a cluster.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_CLUSTER_HOST"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorClusterHostKey),
	},
	{
		FlagName:             "cluster-port",
		ConfigPath:           types.OrchestratorClusterPortKey,
		DefaultValue:         config.Default.Orchestrator.Cluster.Port,
		Description:          `Port to listen for connections from other orchestrators to form a cluster.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_CLUSTER_PORT"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorClusterPortKey),
	},
	{
		FlagName:             "cluster-advertised-address",
		ConfigPath:           types.OrchestratorClusterAdvertiseKey,
		DefaultValue:         config.Default.Orchestrator.Cluster.Advertise,
		Description:          `Address to advertise to other orchestrators to connect to.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_CLUSTER_ADVERTISEADDRESS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorClusterAdvertiseKey),
	},
	{
		FlagName:             "cluster-peers",
		ConfigPath:           types.OrchestratorClusterPeersKey,
		DefaultValue:         config.Default.Orchestrator.Cluster.Peers,
		Description:          `Comma-separated list of other orchestrators to connect to to form a cluster.`,
		EnvironmentVariables: []string{"BACALHAU_NODE_NETWORK_CLUSTER_PEERS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.OrchestratorClusterPeersKey),
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
