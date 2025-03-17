package configflags

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DataDirFlag = []Definition{
	{
		FlagName:             "data-dir",
		ConfigPath:           types.DataDirKey,
		DefaultValue:         config.Default.DataDir,
		Description:          "The filesystem path bacalhau inits or opens a repo in",
		EnvironmentVariables: []string{"BACALHAU_DIR"},
	},
}
