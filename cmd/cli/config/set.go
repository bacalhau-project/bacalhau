package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
)

func newSetCmd() *cobra.Command {
	showCmd := &cobra.Command{
		Use:          "set",
		Args:         cobra.MinimumNArgs(2),
		Short:        "Set a value in the config.",
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo. We need to ensure a repo
			// exists before we can create or modify a config file in it.
			_, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			return setConfig(cmd.PersistentFlags().Lookup("path").Value.String(), args[0], args[1:]...)
		},
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		cfgDir = ""
	}
	configFilePath := filepath.Join(cfgDir, "bacalhau", "config.yaml")
	showCmd.PersistentFlags().String("path", configFilePath, "Optionally provide a path to a config file to use")
	return showCmd
}

func setConfig(cfgFilePath, key string, value ...string) error {
	// get a map of all allowed config keys and their value type, a poor mans schema.
	typ, ok := cfgtypes.AllKeys()[key]
	if !ok {
		return fmt.Errorf("%q is not a valid config key", key)
	}

	v := viper.New()
	v.SetConfigFile(cfgFilePath)
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if typ == reflect.TypeOf(cfgtypes.Duration(0)) {
		duration, err := time.ParseDuration(value[0])
		if err != nil {
			return fmt.Errorf("Failed to parse duration for path %s: %v\n", key, err)
		}
		v.Set(key, duration.String())
	} else if typ == reflect.TypeOf([]string{}) {
		v.Set(key, value)
	} else {
		// Depending on the type, cast and use the value
		switch typ.Kind() {
		case reflect.String:
			v.Set(key, value[0])
		case reflect.Bool:
			parsed, err := strconv.ParseBool(value[0])
			if err != nil {
				return err
			}
			v.Set(key, parsed)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parsed, err := strconv.ParseInt(value[0], 10, 64)
			if err != nil {
				return err
			}
			v.Set(key, parsed)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			parsed, err := strconv.ParseUint(value[0], 10, 64)
			if err != nil {
				return err
			}
			v.Set(key, parsed)
		default:
			// TODO log an error stateing the user will need to manually edit the config file.
			return fmt.Errorf("unsupported type: %v", typ)
		}
	}
	if err := v.WriteConfig(); err != nil {
		return err
	}

	return nil
}
