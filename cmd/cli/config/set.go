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
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}
		viperWriter.Set(key, value)
	case bool:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("setting %q: invalid boolean value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, boolValue)
	case int, int8, int16, int32, int64:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("setting %q: invalid integer value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, intValue)
	case uint, uint8, uint16, uint32, uint64:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("setting %q: invalid integer value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, uintValue)
	case float32, float64:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("setting %q: invalid float value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, floatValue)
	case types.Duration:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		dur, err := time.ParseDuration(value)
		if err != nil {
			return fmt.Errorf("setting %q: invalid duration value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, dur)
	case model.JobSelectionDataLocality:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		dl, err := model.ParseJobSelectionDataLocality(value)
		if err != nil {
			return fmt.Errorf("setting %q: invalid data locality value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, dl)
	case logger.LogMode:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		lm, err := logger.ParseLogMode(value)
		if err != nil {
			return fmt.Errorf("setting %q: invalid log mode value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, lm)
	case types.StorageType:
		value, err := singleValueOrError(values...)
		if err != nil {
			return fmt.Errorf("setting %q: %w", key, err)
		}

		st, err := types.ParseStorageType(value)
		if err != nil {
			return fmt.Errorf("setting %q: invalid storage type value: %q: %w", key, value, err)
		}
		viperWriter.Set(key, st)
	default:
		return fmt.Errorf("unsupported type %T for key: %q", curValue, key)
	}
	return viperWriter.WriteConfig()
}
