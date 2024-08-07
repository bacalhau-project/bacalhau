package cliflags

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const defaultBacalhauDir = ".bacalhau"

// getDefaultRepo determines the appropriate default directory for storing repository data.
// Priority order:
// 1. If the environment variable BACALHAU_DIR is set and non-empty, use it.
// 2. If the environment variable FIL_WALLET_ADDRESS is set, it tries to use ROOT_DIR.
// 3. User's home directory with .bacalhau appended.
// 4. User-specific configuration directory with .bacalhau appended.
// 5. If all above fail, use .bacalhau in the current directory.
// The function logs any errors encountered during the process and always returns a usable path.
func getDefaultRepo() string {
	if repoDir, set := os.LookupEnv("BACALHAU_DIR"); set && repoDir != "" {
		return repoDir
	}

	if userHome, err := os.UserHomeDir(); err == nil {
		return filepath.Join(userHome, defaultBacalhauDir)
	}

	if userDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(userDir, defaultBacalhauDir)
	}

	log.Error().Msg("Failed to determine default repo path. Using current directory.")
	return defaultBacalhauDir
}

type RepoFlag struct {
	Value string
}

// NewRepoFlag creates a new RepoFlag with the default value set
func NewRepoFlag() *RepoFlag {
	return &RepoFlag{Value: getDefaultRepo()}
}

func (rf *RepoFlag) String() string {
	return rf.Value
}

func (rf *RepoFlag) Set(value string) error {
	rf.Value = value
	return nil
}

func (rf *RepoFlag) Type() string {
	return "string"
}
