package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var ServeFlags = []Definition{
	{
		FlagName:     "orchestrator",
		ConfigPath:   types.OrchestratorEnabledKey,
		DefaultValue: config.Default.Orchestrator.Enabled,
		Description:  "When true the orchestrator service will be enabled.",
	},
	{
		FlagName:     "compute",
		ConfigPath:   types.ComputeEnabledKey,
		DefaultValue: config.Default.Compute.Enabled,
		Description:  "When true the compute service will be enabled.",
	},
}
