package configflags

import "github.com/bacalhau-project/bacalhau/pkg/config/types"

var AllowListLocalPathsFlags = []Definition{
	{
		FlagName:     "allow-listed-local-paths",
		ConfigPath:   types.NodeAllowListedLocalPaths,
		DefaultValue: Default.Node.AllowListedLocalPaths,
		Description:  "Local paths that are allowed to be mounted into jobs",
	},
}
