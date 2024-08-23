package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var OrchestratorFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   "Orchestrator.Enabled",
		DefaultValue: types2.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:     "advertised-address",
		ConfigPath:   "Orchestrator.Advertise",
		DefaultValue: types2.Default.Orchestrator.Advertise,
		Description:  `Address to advertise to compute nodes to connect to.`,
	},
	{
		FlagName:     "cluster-listen-address",
		ConfigPath:   "Orchestrator.Cluster.Listen",
		DefaultValue: types2.Default.Orchestrator.Cluster.Listen,
		Description:  `Address to listen on for other orchestrators to connect to..`,
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
}
