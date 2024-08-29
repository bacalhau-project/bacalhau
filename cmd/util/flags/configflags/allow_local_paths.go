package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

var AllowListLocalPathsFlags = []Definition{
	{
		FlagName:     "allow-listed-local-paths",
		ConfigPath:   cfgtypes.ComputeAllowListedLocalPathsKey,
		DefaultValue: cfgtypes.Default.Compute.AllowListedLocalPaths,
		Description:  "Local paths that are allowed to be mounted into jobs",
	},
}
