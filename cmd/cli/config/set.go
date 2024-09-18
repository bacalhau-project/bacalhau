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
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
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
			// Get the value of the --config flag
			configFlag, err := cmd.Flags().GetString("config")
			if err != nil {
				return fmt.Errorf("failed to get config flag: %w", err)
			}

			// If --config flag is set, use it to set the Viper key
			if configFlag != "" {
				viper.Set(cliflags.RootCommandConfigFiles, []string{configFlag})
			}

			var configPath string
			// load configs to get the config file path
			rawConfig, err := util.SetupConfigType(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}

			configPath = rawConfig.ConfigFileUsed()
			if configPath == "" {
				// we fall back to the default config file path $BACALHAU_DIR/config.yaml
				// this requires initializing a new or opening an existing data-dir
				bacalhauConfig, err := util.DecodeBacalhauConfig(rawConfig)
				if err != nil {
					return fmt.Errorf("failed to decode bacalhau config: %w", err)
				}
				_, err = util.SetupRepo(bacalhauConfig)
				if err != nil {
					return fmt.Errorf("failed to setup repo: %w", err)
				}
				configPath = filepath.Join(bacalhauConfig.DataDir, config.DefaultFileName)

				// create the config file if it doesn't exist
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					if err := os.WriteFile(configPath, []byte{}, util2.OS_USER_RWX); err != nil {
						return fmt.Errorf("failed to create default config file %s: %w", configPath, err)
					}
				}
			}

			return setConfig(configPath, args[0], args[1:]...)
		},
		// Provide auto completion for arguments to the `set` command
		ValidArgsFunction: setAutoComplete,
	}

	setCmd.PersistentFlags().String("config", "", "Path to the config file (default is $BACALHAU_DIR/config.yaml)")
	return setCmd
}

func setConfig(cfgFilePath, key string, value ...string) error {
	log.Info().Msgf("Writing config to %s", cfgFilePath)
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
