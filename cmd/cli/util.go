package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
)

const defaultBacalhauDir = ".bacalhau"

// defaultRepo determines an appropriate directory for storing repository data.
// Priority order:
// 1. If the environment variable BACALHAU_DIR is set, use it.
// 2. If the environment variable FIL_WALLET_ADDRESS is set, it tries to use ROOT_DIR.
// 3. The user's home directory.
// 4. User-specific configuration directory.
// 5. User-specific cache directory.
// 6. System's temporary directory.
// Each step logs a warning on failure and proceeds to the next.
// The function returns the chosen directory path or an error if no suitable directory is found.
func defaultRepo() (string, error) {
	// Check known locations for a valid path; error if none found.

	if _, set := os.LookupEnv("BACALHAU_DIR"); set {
		repoDir := os.Getenv("BACALHAU_DIR")
		if repoDir != "" {
			return repoDir, nil
		}
	}

	//If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	//and not a generic environment variable set by the user
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
		log.Warn().Err(err).Msg("Failed to find user home dir. Trying user config directory next.")
	}

	// UserConfigDir provides the root directory for user-specific configuration data.
	// Unix: $XDG_CONFIG_HOME or $HOME/.config, Darwin: $HOME/Library/Application Support,
	// Windows: %AppData%, Plan 9: $home/lib.
	userDir, err := os.UserConfigDir()
	if err == nil {
		return filepath.Join(userDir, defaultBacalhauDir), nil
	} else {
		log.Warn().Err(err).Msg("Failed to find user config dir. Trying user cache directory next.")
	}

	// UserCacheDir provides the root directory for user-specific cached data.
	// Unix: $XDG_CACHE_HOME or $HOME/.cache, Darwin: $HOME/Library/Caches,
	// Windows: %LocalAppData%, Plan 9: $home/lib/cache.
	userCache, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(userCache, defaultBacalhauDir), nil
	} else {
		log.Warn().Err(err).Msg("Failed to find user cache dir. Defaulting to temp directory.")
	}

	// TempDir returns the default temp directory.
	// Unix: $TMPDIR or /tmp, Windows: %TMP%, %TEMP%, %USERPROFILE% or Windows directory, Plan 9: /tmp.
	tempDir := os.TempDir()

	// Check if the directory exists.
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return "", fmt.Errorf("temp directory (%s) does not exist", tempDir)
	} else if err != nil {
		return "", err
	}

	// else we use the temp dir
	return tempDir, nil
}
