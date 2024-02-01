package config

import (
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

type Option func(options *Params)

func WithFileName(name string) Option {
	return func(options *Params) {
		options.FileName = name
	}
}

func WithFileType(ftype string) Option {
	return func(options *Params) {
		options.FileType = ftype
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

func WriteConfigHandler(fileName string) error {
	var cfg types.BacalhauConfig
	if err := viper.Unmarshal(&cfg, configDecoderHook); err != nil {
		return err
	}

	cfgBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	flags := os.O_CREATE | os.O_TRUNC | os.O_WRONLY
	f, err := os.OpenFile(fileName, flags, os.FileMode(0o644)) //nolint:gomnd
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(cfgBytes); err != nil {
		return err
	}

	// read the config we wrote into viper, setting its values as the defaults used for configuration
	return viper.ReadInConfig()
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
