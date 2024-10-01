package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var setExample = templates.Examples(i18n.T(`
bacalhau config set api.host=127.0.0.1
bacalhau config set compute.orchestrators=http://127.0.0.1:1234,http://1.1.1.1:1234
bacalhau config set compute.labels=foo=bar,baz=buz
`))

func newSetCmd() *cobra.Command {
	setCmd := &cobra.Command{
		Use:          "set",
		Args:         cobra.MinimumNArgs(1),
		Short:        "Set a value in the config.",
		Long:         "The 'set' command allows you to modify configuration values in your Bacalhau configuration file.",
		Example:      setExample,
		PreRunE:      hook.ClientPreRunHooks,
		PostRunE:     hook.ClientPostRunHooks,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// load configs to get the config file path
			bacalhauConfig, rawConfig, err := util.SetupConfigs(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup config: %w", err)
			}

			configPath := rawConfig.ConfigFileUsed()
			if configPath == "" {
				// we fall back to the default config file path $BACALHAU_DIR/config.yaml
				// this requires initializing a new or opening an existing data-dir
				_, err = util.SetupRepo(bacalhauConfig)
				if err != nil {
					return fmt.Errorf("failed to setup data dir: %w", err)
				}
				configPath = filepath.Join(bacalhauConfig.DataDir, config.DefaultFileName)
			}

			var (
				key   string
				value []string
			)

			// Check if the first argument is in key=value format
			if strings.Contains(args[0], "=") {
				parts := strings.SplitN(args[0], "=", 2)
				key = parts[0]
				value = []string{parts[1]}
			} else {
				// Fallback to the original behavior: key and value are separate arguments
				if len(args) < 2 {
					return fmt.Errorf("must provide both key and value, or key=value")
				}
				cmd.Println("DEPRECATED: use key=value instead of space-separated key value")
				key = args[0]
				value = args[1:]
			}

			return setConfig(cmd, rawConfig, configPath, key, value...)
		},
		// Provide auto completion for arguments to the `set` command
		ValidArgsFunction: setAutoComplete,
	}

	setCmd.PersistentFlags().VarP(cliflags.NewWriteConfigFlag(), "config", "c", "Path to the config file (default is $BACALHAU_DIR/config.yaml)")
	return setCmd
}

func setConfig(cmd *cobra.Command, cfg *config.Config, cfgFilePath, key string, value ...string) error {
	cmd.Printf("Writing config to %s", cfgFilePath)
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
	if key == types.DataDirKey {
		currentRepoPath := cfg.Get(types.DataDirKey)
		if currentRepoPath != "" {
			if filepath.Join(currentRepoPath.(string), config.DefaultFileName) == cfgFilePath {
				return bacerrors.New("modifying the config key %q within the bacalhau repo config %q is not permitted.",
					types.DataDirKey, cfgFilePath).WithHint("You are free to do so manually, but advised against it.")
			}
		}
		dataDirPath, err := config.ExpandPath(parsed.(string))
		if err != nil {
			return err
		}
		v.Set(key, dataDirPath)
	} else {
		v.Set(key, parsed)
	}
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
