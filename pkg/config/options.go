package config

import (
	"os"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type Option func(options *params)

func WithFileName(name string) Option {
	return func(options *params) {
		options.FileName = name
	}
}

func WithDefaultConfig(cfg types.BacalhauConfig) Option {
	return func(options *params) {
		options.DefaultConfig = cfg
	}
}

func WithFileHandler(handler func(v *viper.Viper, name string) error) Option {
	return func(options *params) {
		options.FileHandler = handler
	}
}

func NoopConfigHandler(v *viper.Viper, filename string) error {
	return nil
}

func ReadConfigHandler(v *viper.Viper, fileName string) error {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		// if the config file doesn't exist that's fine, we will just use default configuration values
		// dictated by the environment
		return nil
	} else if err != nil {
		return err
	}
	// else we will read values set from the config, and accept those over the default values.
	return v.ReadInConfig()
}
