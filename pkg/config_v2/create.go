package config_v2

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func InitConfig(configPath string, configName string, configType string) error {
	// configure viper.
	viper.SetConfigName(configName)
	viper.SetConfigType(configType)
	viper.AddConfigPath(configPath)
	viper.SetEnvPrefix("BACALHAU")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	SetDefaults()
	var cfg BacalhauConfig
	if err := viper.Unmarshal(&cfg, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()); err != nil {
		return err
	}
	// now write the default values to it.
	if err := viper.SafeWriteConfig(); err != nil {
		return fmt.Errorf("viper failed to write config: %w", err)
	}
	if err := viper.ReadInConfig(); err != nil {
		return err
	}
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("viper failed to write config: %w", err)
	}

	// now register env vars
	viper.AutomaticEnv()
	return nil
}
