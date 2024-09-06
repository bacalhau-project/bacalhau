package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	util2 "github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

func newSetCmd() *cobra.Command {
	configFilePath := ".bacalhau"
	setCmd := &cobra.Command{
		Use:   "set",
		Args:  cobra.MinimumNArgs(2),
		Short: "Set a value in the config.",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := os.UserConfigDir()
			if err != nil {
				cfgDir = ""
			}
			configFileDirPath := filepath.Join(cfgDir, "bacalhau")
			if err := os.MkdirAll(configFileDirPath, util2.OS_USER_RWX); err != nil {
				return fmt.Errorf("failed to create bacalhau config at %s: %w", filepath.Join(configFileDirPath, "config.yaml"))
			}
			configFilePath = filepath.Join(configFileDirPath, "config.yaml")
			return nil
		},
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo. We need to ensure a repo
			// exists before we can create or modify a config file in it.
			_, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			return setConfig(cmd.PersistentFlags().Lookup("config").Value.String(), args[0], args[1:]...)
		},
		// provide auto completion for arguments to the `set` command
		ValidArgsFunction: setAutoComplete,
	}
	setCmd.PersistentFlags().String("config", configFilePath, "Optionally provide a path to a config file to use")
	return setCmd
}

func setConfig(cfgFilePath, key string, value ...string) error {
	v := viper.New()
	v.SetConfigFile(cfgFilePath)
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	parsed, err := types.CastConfigValueForKey(key, value)
	if err != nil {
		return err
	}
	v.Set(key, parsed)
	if err := v.WriteConfig(); err != nil {
		return err
	}

	return nil
}

func setAutoComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
