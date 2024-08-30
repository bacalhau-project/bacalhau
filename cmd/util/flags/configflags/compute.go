package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var ComputeFlags = []Definition{
	{
		FlagName:     "compute",
		ConfigPath:   types.ComputeEnabledKey,
		DefaultValue: types.Default.Compute.Enabled,
		Description:  "When true the compute service will be enabled.",
	},
}
