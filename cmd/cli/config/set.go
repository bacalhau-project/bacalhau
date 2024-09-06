package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	util2 "github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

func newSetCmd() *cobra.Command {
	setCmd := &cobra.Command{
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
			return setConfig(cmd.PersistentFlags().Lookup("config").Value.String(), args[0], args[1:]...)
		},
		// provide auto completion for arguments to the `set` command
		ValidArgsFunction: setAutoComplete,
	}

	bacalhauCfgDir := "bacalhau"
	bacalhauCfgFile := config.DefaultFileName

	usrCfgDir, err := os.UserConfigDir()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to find user-specific configuration directory. Using current directory to write config.")
	} else {
		bacalhauCfgDir = filepath.Join(usrCfgDir, bacalhauCfgDir)
		if err := os.MkdirAll(bacalhauCfgDir, util2.OS_USER_RWX); err != nil {
			// This means we failed to create a directory either in the current directory, or the user config dir
			// indicating a some-what serious misconfiguration of the system. We panic here to provide as much
			// detail as possible.
			log.Panic().Err(err).Msgf("Failed to create bacalhau configuration directory: %s", bacalhauCfgDir)
		}
		bacalhauCfgFile = filepath.Join(bacalhauCfgDir, bacalhauCfgFile)
	}

	setCmd.PersistentFlags().String("config", bacalhauCfgFile, "Path to the config file")
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
	if err := v.WriteConfigAs(cfgFilePath); err != nil {
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
