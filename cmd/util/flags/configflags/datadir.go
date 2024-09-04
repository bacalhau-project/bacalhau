package configflags

import (
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

var DataDirFlag = []Definition{
	{
		FlagName:             "repo",
		ConfigPath:           types.DataDirKey,
		DefaultValue:         getDefaultRepo(),
		Description:          "The filesystem path bacalhau inits or opens a repo in",
		EnvironmentVariables: []string{"BACALHAU_DIR"},
		Deprecated:           true,
		DeprecatedMessage:    "Use --data-dir=<path> to set this configuration",
	},
	{
		FlagName:             "data-dir",
		ConfigPath:           types.DataDirKey,
		DefaultValue:         getDefaultRepo(),
		Description:          "The filesystem path bacalhau inits or opens a repo in",
		EnvironmentVariables: []string{"BACALHAU_DIR"},
	},
}

const defaultBacalhauDir = ".bacalhau"

// getDefaultRepo determines the appropriate default directory for storing repository data.
// Priority order:
// 1. If the environment variable BACALHAU_DIR is set and non-empty, use it.
// 2. User's home directory with .bacalhau appended.
// 3. If all above fail, use .bacalhau in the current directory.
func getDefaultRepo() string {
	if userHome, err := os.UserHomeDir(); err == nil {
		return filepath.Join(userHome, defaultBacalhauDir)
	}

	return defaultBacalhauDir
}
