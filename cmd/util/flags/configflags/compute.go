package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var ComputeFlags = []Definition{
	{
		FlagName:     "compute",
		ConfigPath:   "Compute.Enabled",
		DefaultValue: types2.Default.Compute.Enabled,
		Description:  "When true the compute service will be enabled.",
	},
	{
		FlagName:     "orchestrators",
		ConfigPath:   "Compute.Orchestrators",
		DefaultValue: types2.Default.Compute.Orchestrators,
		Description:  `Comma-separated list of orchestrators to connect to. Applies to compute nodes.`,
	},
}
