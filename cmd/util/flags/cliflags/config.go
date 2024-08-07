package cliflags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// ConfigAutoComplete provides auto-completion suggestions for configuration keys.
func ConfigAutoComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string

	// Iterate over the ConfigDescriptions map to find matching keys
	for key, description := range types.ConfigDescriptions {
		if strings.HasPrefix(key, toComplete) {
			completion := fmt.Sprintf("%s\t%s", key, description)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace
}

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
			cfgKey := tokens[0]
			cfgValue := tokens[1]
			return setIfValid(viper.GetViper(), cfgKey, cfgValue)
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
		return setIfValid(viper.GetViper(), cf.Value, true)
	}
	return nil
}

func setIfValid(v *viper.Viper, key string, value any) error {
	if _, ok := types.ConfigDescriptions[strings.ToLower(key)]; !ok {
		return fmt.Errorf("no config key matching %q", key)
	}
	v.Set(key, value)
	return nil
}
