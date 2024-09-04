package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DataDirFlag = []Definition{
	{
		FlagName:             "repo",
		ConfigPath:           types.DataDirKey,
		DefaultValue:         types.Default.DataDir,
		Description:          "The filesystem path bacalhau inits or opens a repo in",
		EnvironmentVariables: []string{"BACALHAU_DIR"},
		Deprecated:           true,
		DeprecatedMessage:    "Use --data-dir=<path> to set this configuration",
	},
	{
		FlagName:             "data-dir",
		ConfigPath:           types.DataDirKey,
		DefaultValue:         types.Default.DataDir,
		Description:          "The filesystem path bacalhau inits or opens a repo in",
		EnvironmentVariables: []string{"BACALHAU_DIR"},
	},
}
