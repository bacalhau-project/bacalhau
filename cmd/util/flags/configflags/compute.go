package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var ComputeFlags = []Definition{
	{
		FlagName:     "compute",
		ConfigPath:   cfgtypes.ComputeEnabledKey,
		DefaultValue: cfgtypes.Default.Compute.Enabled,
		Description:  "When true the compute service will be enabled.",
	},
}
