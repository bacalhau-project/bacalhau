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

func NoopConfigHandler(filename string) error {
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
