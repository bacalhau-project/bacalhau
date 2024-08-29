package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var ComputeFlags = []Definition{
	{
		FlagName:     "compute",
		ConfigPath:   types2.ComputeEnabledKey,
		DefaultValue: types2.Default.Compute.Enabled,
		Description:  "When true the compute service will be enabled.",
	},
}
