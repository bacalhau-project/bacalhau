package config

import (
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func SetIntallationID(path string) {
	viper.Set(types.UserInstallationID, path)
}

func SetUpdateCheckStatePath(path string) {
	viper.Set(types.UpdateCheckStatePath, path)
}

// SetDefault sets the default value for the configuration.
// Default only used when no value is provided by the user via an explicit call to Set, flag, config file or ENV.
func SetDefault(config types.BacalhauConfig) error {
	types.SetDefaults(config)
	return nil
}

// Set sets the configuration value.
// Will be used instead of values obtained via flags, config file, ENV, default.
// Useful for testing.
func Set(config types.BacalhauConfig) error {
	types.Set(config)
	return nil
}
