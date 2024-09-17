package repo

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func ReadInstallationID() string {
	var idFile string
	os.UserCacheDir()
	switch runtime.GOOS {
	case "linux":
		configDir := os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			configDir = filepath.Join(os.Getenv("HOME"), ".config")
		}
		idFile = filepath.Join(configDir, "bacalhau", "installation_id")
	case "darwin":
		idFile = filepath.Join(os.Getenv("HOME"), "Library", "Preferences", "com.bacalhau.installation_id")
	case "windows":
		appData := os.Getenv("APPDATA")
		idFile = filepath.Join(appData, "bacalhau", "installation_id")
	default:
		configDir := filepath.Join(os.Getenv("HOME"), ".config")
		idFile = filepath.Join(configDir, "bacalhau", "installation_id")
	}

	idBytes, err := os.ReadFile(idFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(idBytes))
}
