package config

import (
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/spf13/viper"
)

type Option func(options *Params)

func WithFileName(name string) Option {
	return func(options *Params) {
		options.FileName = name
	}
}

func WithDefaultConfig(cfg types.BacalhauConfig) Option {
	return func(options *Params) {
		options.DefaultConfig = cfg
	}
}

func WithFileHandler(handler func(name string) error) Option {
	return func(options *Params) {
		options.FileHandler = handler
	}
}

func WithPostConfigHandler(handler func(name string, cfg types.BacalhauConfig) error) Option {
	return func(options *Params) {
		options.PostConfigHandler = handler
	}
}

func NoopConfigHandler(filename string) error {
	return nil
}

func NoopPostConfigHandler(filename string, cfg types.BacalhauConfig) error {
	return nil
}

func ReadConfigHandler(fileName string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// if the config file doesn't exist that's fine, we will just use default configuration values
		// dictated by the environment
		return nil
	} else if err != nil {
		return err
	}
	// else we will read values set from the config, and accept those over the default values.
	return viper.ReadInConfig()
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

	// any value that is not set in the persisted config, and the resolved value does not match the
	// default value, we will set it in the persisted config.
	var doWrite bool
	emptyStoreConfig := types.JobStoreConfig{}
	if fileCfg.Node.Compute.ExecutionStore == emptyStoreConfig {
		viperWriter.Set(types.NodeComputeExecutionStore, resolvedCfg.Node.Compute.ExecutionStore)
		doWrite = true
	}
	if fileCfg.Node.Requester.JobStore == emptyStoreConfig {
		viperWriter.Set(types.NodeRequesterJobStore, resolvedCfg.Node.Requester.JobStore)
		doWrite = true
	}
	if doWrite {
		return viperWriter.WriteConfig()
	}
	return nil
}
