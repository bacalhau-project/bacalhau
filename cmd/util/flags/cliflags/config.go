package cliflags

import (
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
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
			completion := fmt.Sprintf("%s=\t%s", key, description)
			completions = append(completions, completion)
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace
}

const RootCommandConfigFiles = "Root.Command.Config.Files"
const RootCommandConfigValues = "Root.Command.Config.Values"
const RootCommandConfigFlags = "Root.Command.Config.Flags"

type ConfigFlag struct {
	Value       string
	isWriteMode bool
}

func NewConfigFlag() *ConfigFlag {
	viper.Set(RootCommandConfigValues, new(map[string]any))
	viper.Set(RootCommandConfigFiles, make([]string, 0))
	return &ConfigFlag{isWriteMode: false}
}

func NewWriteConfigFlag() *ConfigFlag {
	viper.Set(RootCommandConfigFiles, "")
	return &ConfigFlag{isWriteMode: true}
}

func (cf *ConfigFlag) String() string {
	return cf.Value
}

func (cf *ConfigFlag) Set(value string) error {
	if cf.isWriteMode {
		// Check if a config file is already set in Viper
		if viper.GetString(RootCommandConfigFiles) != "" {
			return fmt.Errorf("single config file can be set")
		}
	}
	cf.Value = value
	return cf.Parse()
}

func (cf *ConfigFlag) Type() string {
	return "string"
}

func (cf *ConfigFlag) Parse() error {
	if cf.isWriteMode {
		return cf.parseWriteMode()
	}
	return cf.parseReadMode()
}

func (cf *ConfigFlag) parseWriteMode() error {
	if !cf.isConfigFile() {
		return fmt.Errorf("config file must end with '.yaml' or '.yml'")
	}

	if err := validateFile(cf.Value); err != nil {
		return err
	}

	viper.Set(RootCommandConfigFiles, cf.Value)
	return nil
}

func (cf *ConfigFlag) parseReadMode() error {
	if strings.Contains(cf.Value, "=") {
		// Handle key-value pair
		return cf.parseKeyValue()
	} else if cf.isConfigFile() {
		// Handle YAML file
		return cf.addConfigFile()
	} else {
		// Handle dot separated path with boolean value
		return setIfValid(viper.GetViper(), cf.Value, true)
	}
}

func (cf *ConfigFlag) parseKeyValue() error {
	tokens := strings.SplitN(cf.Value, "=", 2)
	if len(tokens) != 2 {
		return fmt.Errorf("config flag value %s is invalid", cf.Value)
	}
	return setIfValid(viper.GetViper(), tokens[0], tokens[1])
}

func (cf *ConfigFlag) addConfigFile() error {
	if err := validateFile(cf.Value); err != nil {
		return err
	}

	configFiles := viper.GetStringSlice(RootCommandConfigFiles)
	configFiles = append(configFiles, cf.Value)
	viper.Set(RootCommandConfigFiles, configFiles)
	return nil
}

func validateFile(filePath string) error {
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("the specified configuration file %q doesn't exist", filePath)
		}
		return fmt.Errorf("the specified configuration file %q cannot be read: %w", filePath, err)
	}
	if stat.IsDir() {
		return fmt.Errorf("the specified configuration file %q is a directory, must be a file", filePath)
	}
	if stat.Size() == 0 {
		log.Warn().Msgf("the specified configuration file is empty and ineffectual")
	}
	return nil
}

func setIfValid(v *viper.Viper, key string, value any) error {
	key = strings.ToLower(key)
	_, ok := types.AllKeys()[key]
	if !ok {
		if _, err := os.Stat(key); err == nil {
			return fmt.Errorf("config files must end in suffix '.yaml' or '.yml'")
		}
		return fmt.Errorf("no config key matching %q run 'bacalhau config list' for a list of valid keys", key)
	}
	parsed, err := types.CastConfigValueForKey(key, value)
	if err != nil {
		return err
	}
	configMap := v.GetStringMap(RootCommandConfigValues)
	configMap[key] = parsed
	v.Set(RootCommandConfigValues, configMap)
	return nil
}

func (cf *ConfigFlag) isConfigFile() bool {
	return strings.HasSuffix(cf.Value, ".yaml") || strings.HasSuffix(cf.Value, ".yml")
}
