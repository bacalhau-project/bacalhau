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

	"github.com/bacalhau-project/bacalhau/cmd/util"
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
		PreRunE:  util.ClientPreRunHooks,
		PostRunE: util.ClientPostRunHooks,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setConfig(args[0], args[1:]...)
		},
	}
	return showCmd
}

func setConfig(key string, values ...string) error {
	// remove all spaces and make lowercase
	key = sanitizeKey(key)
	// get the default viper schema
	viperSchema := NewViperWithDefaultConfig(config.ForEnvironment())
	// get a list of all valid configuration keys, same list as returned by `config list`
	liveKeys := viperSchema.AllKeys()
	if !slices.Contains(liveKeys, key) {
		return fmt.Errorf("invalid configuration key %q: not found", key)
	}

	// there may me a config file present, we'll write to that if it exists.
	configFile := viper.ConfigFileUsed()

	// we create a new viper instance that we'll use to set values on and update the config.
	viperWriter := viper.New()
	viperWriter.SetTypeByDefaultValue(true)
	if configFile == "" {
		// if there isn't a config file, we'll assume we add it to the current repo.
		configFile = filepath.Join(viper.GetString("repo"), "config.yaml")
	}
	viperWriter.SetConfigFile(configFile)
	// the instance has read in a copy of the config file
	if err := viperWriter.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	//calling `Get` on this instance will return a default value from the config structure that we can type assert on.
	curValue := viperSchema.Get(key)

	// we need special handling for the nodetype since its just a string, but carries implicit requirements in the config.
	if strings.EqualFold(types.NodeType, key) {
		if !validNodeTypes(values) {
			return fmt.Errorf("setting: %q, invalid node type value: %q, must be one of: 'requester' 'compute' 'requester compute'", key, values)
		}
	}

	type parserFunc func(string) (any, error)
	var parser parserFunc

	switch curValue.(type) {
	case []string:
		viperWriter.Set(key, values)
	case map[string]string:
		sts, err := parseStringSliceToMap(values)
		if err != nil {
			return err
		}
		viperWriter.Set(key, sts)
	case string:
		parser = func(s string) (any, error) { return s, nil } // identity by default
	case bool:
		parser = func(s string) (any, error) { return strconv.ParseBool(s) }
	case int, int8, int16, int32, int64:
		parser = func(s string) (any, error) { return strconv.ParseInt(s, 10, 64) }
	case uint, uint8, uint16, uint32, uint64:
		parser = func(s string) (any, error) { return strconv.ParseUint(s, 10, 64) }
	case float32, float64:
		parser = func(s string) (any, error) { return strconv.ParseFloat(s, 10) }
	case types.Duration, time.Duration:
		parser = func(s string) (any, error) { return time.ParseDuration(s) }
	case model.JobSelectionDataLocality:
		parser = func(s string) (any, error) { return model.ParseJobSelectionDataLocality(s) }
	case logger.LogMode:
		parser = func(s string) (any, error) { return logger.ParseLogMode(s) }
	case types.StorageType:
		parser = func(s string) (any, error) { return types.ParseStorageType(s) }
	default:
		return fmt.Errorf("unsupported type %T for key: %q", curValue, key)
	}

	if parser != nil {
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		configValue, err := parser(value)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}
		viperWriter.Set(key, configValue)
	}

	return viperWriter.WriteConfig()
}
