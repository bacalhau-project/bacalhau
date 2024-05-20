package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func newSetCmd() *cobra.Command {
	showCmd := &cobra.Command{
		Use:      "set",
		Args:     cobra.MinimumNArgs(2),
		Short:    "Set a value in the config.",
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo. We need to ensure a repo
			// exists before we can create or modify a config file in it.
			_, err := util.SetupRepoConfig()
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			return setConfig(args[0], args[1:]...)
		},
	}
	return showCmd
}

func currentValue(key string) (interface{}, error) {
	// get the default viper schema
	viperSchema := NewViperWithDefaultConfig(config.ForEnvironment())
	// get a list of all valid configuration keys, same list as returned by `config list`
	liveKeys := viperSchema.AllKeys()
	if !slices.Contains(liveKeys, key) {
		return nil, fmt.Errorf("invalid configuration key %q: not found", key)
	}

	// calling `Get` on this instance will return a default value from the config structure that we can type assert on.
	return viperSchema.Get(key), nil
}

func getWriter(configFile string) (*viper.Viper, error) {
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	viperWriter.SetConfigFile(configFile)
	// the instance has read in a copy of the config file
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	return viperWriter, nil
}

func setConfig(key string, values ...string) error {
	// remove all spaces and make lowercase
	key = sanitizeKey(key)

	// we need special handling for the nodetype since its just a string, but carries implicit requirements in the config.
	if strings.EqualFold(types.NodeType, key) {
		if !validNodeTypes(values) {
			return fmt.Errorf("setting: %q, invalid node type value: %q, must be one of: 'requester' 'compute' 'requester compute'", key, values)
		}
	}

	curValue, err := currentValue(key)
	if err != nil {
		return err
	}

	// there may me a config file present, we'll write to that if it exists.
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		// if there isn't a config file, we'll assume we add it to the current repo.
		configFile = filepath.Join(viper.GetString("repo"), "config.yaml")
	}

	viperWriter, err := getWriter(configFile)
	if err != nil {
		return err
	}

	switch curValue.(type) {
	case []string:
		viperWriter.Set(key, values)
	case map[string]string:
		sts, err := parse.StringSliceToMap(values)
		if err != nil {
			return err
		}
		viperWriter.Set(key, sts)
	case map[string]types.AuthenticatorConfig:
		cfg := struct {
			Method string                    `yaml:"Method"`
			Policy types.AuthenticatorConfig `yaml:"Policy"`
		}{}
		if err := yaml.Unmarshal([]byte(values[0]), &cfg); err != nil {
			return err
		}
		methodNamePath := fmt.Sprintf("%s.%s", types.AuthMethods, cfg.Method)
		viperWriter.Set(methodNamePath, cfg.Policy)
	}

	parser, err := getParser(curValue, key)
	if parser == nil || err != nil {
		return viperWriter.WriteConfig()
	}

	value, err := singleValueOrError(values...)
	if err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}

	configValue, err := parser(value)
	if err != nil {
		return fmt.Errorf("setting %q: %w", key, err)
	}
	viperWriter.Set(key, configValue)
	return viperWriter.WriteConfig()
}

type parserFunc func(string) (any, error)

func getParser(curValue interface{}, key string) (parserFunc, error) {
	var parser parserFunc

	switch curValue.(type) {
	case string:
		parser = func(s string) (any, error) { return s, nil } // identity by default
	case bool:
		parser = func(s string) (any, error) { return strconv.ParseBool(s) }
	case int, int8, int16, int32, int64:
		parser = func(s string) (any, error) { return strconv.ParseInt(s, 10, 64) }
	case uint, uint8, uint16, uint32, uint64:
		parser = func(s string) (any, error) { return strconv.ParseUint(s, 10, 64) }
	case float32, float64:
		parser = func(s string) (any, error) { return strconv.ParseFloat(s, 64) }
	case types.Duration, time.Duration:
		parser = func(s string) (any, error) { return time.ParseDuration(s) }
	case model.JobSelectionDataLocality:
		parser = func(s string) (any, error) { return model.ParseJobSelectionDataLocality(s) }
	case logger.LogMode:
		parser = func(s string) (any, error) { return logger.ParseLogMode(s) }
	case types.StorageType:
		parser = func(s string) (any, error) { return types.ParseStorageType(s) }
	default:
		return nil, fmt.Errorf("unsupported type %T for key: %q", curValue, key)
	}

	return parser, nil
}
