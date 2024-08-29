package cliflags

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

// ConfigAutoComplete provides auto-completion suggestions for configuration keys.
func ConfigAutoComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string

	// Iterate over the ConfigDescriptions map to find matching keys
	for key, description := range cfgtypes.ConfigDescriptions {
		if strings.HasPrefix(key, toComplete) {
			completion := fmt.Sprintf("%s\t%s", key, description)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace
}

func NewConfigFlag() *ConfigFlag {
	viper.Set(RootCommandConfigValues, new(map[string]any))
	viper.Set(RootCommandConfigFiles, make([]string, 0))
	return &ConfigFlag{}
}

const RootCommandConfigFiles = "Root.Command.Config.Files"
const RootCommandConfigValues = "Root.Command.Config.Values"
const RootCommandConfigFlags = "Root.Command.Config.Flags"

type Something struct {
	ConfigFiles  []string
	ConfigParams map[string]any
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
		configFiles := viper.GetStringSlice(RootCommandConfigFiles)
		configFiles = append(configFiles, cf.Value)
		if stat, err := os.Stat(cf.Value); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("the specified configuration file %q doesn't exist", cf.Value)
			}
			return fmt.Errorf("the specified configuration file %q cannot be read: %w", cf.Value, err)
		} else if stat.IsDir() {
			return fmt.Errorf("the specified configuration file %q is a directory, must be a file", cf.Value)
		} else if stat.Size() == 0 {
			log.Warn().Msgf("the specified configuration file is empty and ineffectual")
		}
		viper.Set(RootCommandConfigFiles, configFiles)
	} else {
		// Handle dot separated path with boolean value
		return setIfValid(viper.GetViper(), cf.Value, true)
	}
	return nil
}

func setIfValid(v *viper.Viper, key string, value any) error {
	if _, ok := cfgtypes.ConfigDescriptions[strings.ToLower(key)]; !ok {
		if _, err := os.Stat(key); err == nil {
			return fmt.Errorf("config files must end in suffix '.yaml' or '.yml'")
		}
		return fmt.Errorf("no config key matching %q run 'bacalhau config list' for a list of valid keys", key)
	}
	configMap := v.GetStringMap(RootCommandConfigValues)
	configMap[key] = value
	v.Set(RootCommandConfigValues, configMap)
	return nil
}
