package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const defaultBacalhauDir = ".bacalhau"

// defaultRepo determines the appropriate default directory for storing repository data.
// If a user provides the `--repo` flag the value returned from this function will be overridden.
// Priority order:
// 1. If the environment variable BACALHAU_DIR is set, use it.
// 2. If the environment variable FIL_WALLET_ADDRESS is set, it tries to use ROOT_DIR.
// 2. User-specific configuration directory.
// 4. The user's home directory.
// The function returns the chosen directory path or an error if no suitable directory can be determined.
func defaultRepo() (string, error) {
	if _, set := os.LookupEnv("BACALHAU_DIR"); set {
		repoDir := os.Getenv("BACALHAU_DIR")
		if repoDir != "" {
			return repoDir, nil
		}
	}

	// If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	// and not a generic environment variable set by the user
	if _, set := os.LookupEnv("FIL_WALLET_ADDRESS"); set {
		repoDir := os.Getenv("ROOT_DIR")
		if repoDir != "" {
			log.Debug().Str("repo", repoDir).Msg("using station ROOT_DIR as bacalhau repo")
			return repoDir, nil
		}
	}

	// UserHomeDir gets the user's home directory.
	// On Unix/macOS: $HOME, Windows: %USERPROFILE%, Plan 9: $home.
	userHome, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(userHome, defaultBacalhauDir), nil
	} else {
		log.Debug().Err(err).Msg("Failed to find user home dir. Trying user config directory next.")
	}

	// UserConfigDir provides the root directory for user-specific configuration data.
	// Unix: $XDG_CONFIG_HOME or $HOME/.config, Darwin: $HOME/Library/Application Support,
	// Windows: %AppData%, Plan 9: $home/lib.
	userDir, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(userDir, defaultBacalhauDir), nil
	} else {
		log.Debug().Err(err).Msg("Failed to find user config dir.")
	}

	return "", fmt.Errorf("failed to determine default repo path from '$BACALHAU_DIR', '$HOME', or '$XDG_CONFIG_HOME'")
}
