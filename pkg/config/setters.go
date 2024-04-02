package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func SetIntallationID(path string) {
	SetValue(types.UserInstallationID, path)
}

func SetUpdateCheckStatePath(path string) {
	SetValue(types.UpdateCheckStatePath, path)
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

// SetValue sets the configuration value.
// This value won't be persisted in the config file.
// Will be used instead of values obtained via flags, config file, ENV, default.
func SetValue(key string, value interface{}) {
	viper.Set(key, value)
}

// WritePersistedConfigs will write certain values from the resolved config to the persisted config.
// These include fields for configurations that must not change between version updates, such as the
// execution store and job store paths, in case we change their default values in future updates.
func WritePersistedConfigs(configFile string, resolvedCfg types.BacalhauConfig) error {
	// a viper config instance that is only based on the config file.
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	viperWriter.SetConfigFile(configFile)

	// read existing config if it exists.
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	var fileCfg types.BacalhauConfig
	if err := viperWriter.Unmarshal(&fileCfg, DecoderHook); err != nil {
		return err
	}

	// check if any of the values that we want to write are not set in the config file.
	var doWrite bool
	var logMessage strings.Builder
	set := func(key string, value interface{}) {
		viperWriter.Set(key, value)
		logMessage.WriteString(fmt.Sprintf("\n%s:\t%v", key, value))
		doWrite = true
	}
	emptyStoreConfig := types.JobStoreConfig{}
	if fileCfg.Node.Compute.ExecutionStore == emptyStoreConfig {
		set(types.NodeComputeExecutionStore, resolvedCfg.Node.Compute.ExecutionStore)
	}
	if fileCfg.Node.Requester.JobStore == emptyStoreConfig {
		set(types.NodeRequesterJobStore, resolvedCfg.Node.Requester.JobStore)
	}
	if fileCfg.Node.Name == "" && resolvedCfg.Node.Name != "" {
		set(types.NodeName, resolvedCfg.Node.Name)
	}
	if doWrite {
		log.Info().Msgf("Writing to config file %s:%s", configFile, logMessage.String())
		return viperWriter.WriteConfig()
	}
	return nil
}
