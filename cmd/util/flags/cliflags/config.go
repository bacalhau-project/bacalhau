package cliflags

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

const errComponent = "Config"

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
			return bacerrors.New("single config file can be set").
				WithComponent(errComponent).
				WithCode(bacerrors.ValidationError)
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
		return bacerrors.New("config file must end with '.yaml' or '.yml'").
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
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
		key := strings.ToLower(cf.Value)
		if err := validateKey(key); err != nil {
			return err
		}
		typ := types.AllKeys()[key]
		if typ.Kind() == reflect.Bool {
			return setIfValid(viper.GetViper(), cf.Value, true)
		} else {
			return bacerrors.Newf("config flag key requires a value.\n"+
				"To correct this provide a value to the config, e.g. --config %s=<value>", cf.Value).
				WithComponent(errComponent).
				WithCode(bacerrors.ValidationError)
		}
	}
}

func (cf *ConfigFlag) parseKeyValue() error {
	tokens := strings.SplitN(cf.Value, "=", 2)
	if len(tokens) != 2 {
		return bacerrors.Newf("invalid config flag value: %s", cf.Value).
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
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
			return bacerrors.Newf("specified configuration file doesn't exist: %s", filePath).
				WithComponent(errComponent).
				WithCode(bacerrors.NotFoundError)
		}
		return bacerrors.Wrapf(err, "specified configuration file cannot be read: %s", filePath).
			WithComponent(errComponent).
			WithCode(bacerrors.IOError)
	}
	if stat.IsDir() {
		return bacerrors.Newf("specified configuration file is a directory, must be a file: %s", filePath).
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
	}
	if stat.Size() == 0 {
		log.Warn().Msgf("the specified configuration file is empty and ineffectual")
	}
	return nil
}

func validateKey(key string) error {
	key = strings.ToLower(key)
	_, ok := types.AllKeys()[key]
	if !ok {
		if _, err := os.Stat(key); err == nil {
			return bacerrors.New("config files must end in suffix '.yaml' or '.yml'").
				WithCode(bacerrors.ValidationError).
				WithComponent(errComponent)
		}
		return bacerrors.Newf("no matching config key. Run '%s config list' for a list of valid keys", os.Args[0]).
			WithCode(bacerrors.ValidationError).
			WithComponent(errComponent)
	}
	return nil
}

func setIfValid(v *viper.Viper, key string, value any) error {
	if err := validateKey(key); err != nil {
		return err
	}
	parsed, err := types.CastConfigValueForKey(key, value)
	if err != nil {
		return bacerrors.Wrapf(err, "failed to cast config value for key: %s, value: %v", key, value).
			WithComponent(errComponent).
			WithCode(bacerrors.ValidationError)
	}
	configMap := v.GetStringMap(RootCommandConfigValues)
	configMap[key] = parsed
	v.Set(RootCommandConfigValues, configMap)
	return nil
}

func (cf *ConfigFlag) isConfigFile() bool {
	return strings.HasSuffix(cf.Value, ".yaml") || strings.HasSuffix(cf.Value, ".yml")
}
