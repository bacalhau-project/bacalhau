package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var AllowListLocalPathsFlags = []Definition{
	{
		FlagName:             "allow-listed-local-paths",
		ConfigPath:           types.ComputeAllowListedLocalPathsKey,
		DefaultValue:         config.Default.Compute.AllowListedLocalPaths,
		Description:          "Local paths that are allowed to be mounted into jobs",
		EnvironmentVariables: []string{"BACALHAU_NODE_ALLOWLISTEDLOCALPATHS"},
		Deprecated:           true,
		DeprecatedMessage:    makeDeprecationMessage(types.ComputeAllowListedLocalPathsKey),
	},
}
