package configflags

import (
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

var AllowListLocalPathsFlags = []Definition{
	{
		FlagName:     "allow-listed-local-paths",
		ConfigPath:   types2.ComputeAllowListedLocalPathsKey,
		DefaultValue: types2.Default.Compute.AllowListedLocalPaths,
		Description:  "Local paths that are allowed to be mounted into jobs",
	},
}
