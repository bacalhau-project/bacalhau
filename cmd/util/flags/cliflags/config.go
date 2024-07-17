package cliflags

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func NewConfigFlag() *ConfigFlag {
	return &ConfigFlag{}
}

type ConfigFlag struct {
	Value string
}

func (cf *ConfigFlag) String() string {
	return cf.Value
}

func (cf *ConfigFlag) Set(value string) error {
	cf.Value = value
	return cf.Parse()
}

func (cf *ConfigFlag) Type() string {
	return "string"
}

func (cf *ConfigFlag) Parse() error {
	if strings.Contains(cf.Value, "=") {
		// Handle key-value pair
		tokens := strings.SplitN(cf.Value, "=", 2)
		if len(tokens) == 2 {
			viper.Set(tokens[0], tokens[1])
		} else {
			return fmt.Errorf("config flag value %s is invalid", cf.Value)
		}
	} else if strings.HasSuffix(cf.Value, ".yaml") || strings.HasSuffix(cf.Value, ".yml") {
		// Handle YAML file
		viper.SetConfigFile(cf.Value)
		if err := viper.MergeInConfig(); err != nil {
			return fmt.Errorf("error reading config file (%s): %w", cf.Value, err)
		}
	} else {
		// Handle dot separated path with boolean value
		viper.Set(cf.Value, true)
	}
	return nil
}
