package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog/log"
)

// InstallationIDFile is the name of the file storing the installation ID.
const InstallationIDFile = "installation_id"

// GlobalConfig defines the interface for accessing global configuration settings.
// The interface is used to abstract the configuration directory and installation ID
// and allows for easy testing, such as testing migration logic.
type GlobalConfig interface {
	// InstallationID returns the unique identifier for this installation, if available.
	InstallationID() string
	// ConfigDir returns the path to the configuration directory.
	ConfigDir() string
}

// DefaultGlobalConfig is the default implementation of GlobalConfig.
var DefaultGlobalConfig GlobalConfig = &realGlobalConfig{}

// realGlobalConfig is the concrete implementation of GlobalConfig.
type realGlobalConfig struct{}

// ConfigDir returns the path to the Bacalhau configuration directory.
// It respects the XDG Base Directory Specification on Unix-like systems
// and uses the appropriate directory on Windows.
func (r *realGlobalConfig) ConfigDir() string {
	var baseDir string
	switch runtime.GOOS {
	case "linux", "darwin":
		baseDir = os.Getenv("XDG_CONFIG_HOME")
		if baseDir == "" {
			baseDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
	case "windows":
		baseDir = os.Getenv("APPDATA")
	default:
		baseDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(baseDir, "bacalhau")
}

// InstallationID reads and returns the installation ID from the config file.
// If the file doesn't exist or can't be read, it returns an empty string.
func (r *realGlobalConfig) InstallationID() string {
	idFile := filepath.Join(r.ConfigDir(), InstallationIDFile)
	idBytes, err := os.ReadFile(idFile) //nolint:gosec // G304: idFile from repo path, application controlled
	if err != nil {
		if !os.IsNotExist(err) {
			log.Debug().Err(err).Msg("Failed to read installation ID file")
		}
		return ""
	}
	return strings.TrimSpace(string(idBytes))
}

// InstallationID is a convenience function that returns the installation ID
// using the DefaultGlobalConfig.
func InstallationID() string {
	return DefaultGlobalConfig.InstallationID()
}
