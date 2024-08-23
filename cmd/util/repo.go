package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/configv2"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

func SetupRepoConfig(cmd *cobra.Command) (types2.Bacalhau, error) {
	cfg, err := SetupConfig(cmd)
	if err != nil {
		return types2.Bacalhau{}, err
	}
	// create or open the bacalhau repo and load the config
	r, err := SetupRepo(cfg)
	if err != nil {
		return types2.Bacalhau{}, fmt.Errorf("failed to reconcile repo: %w", err)
	}

	// TODO(forrest): we need to start this hook somewhere else as not all CLI methods call this parent method.
	hook.StartUpdateCheck(cmd, cfg, r)

	return cfg, nil
}

func SetupRepo(cfg types2.Bacalhau) (*repo.FsRepo, error) {
	// create or open the bacalhau repo and load the config
	r, err := setup.SetupBacalhauRepo(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile repo: %w", err)
	}
	return r, nil
}

func SetupConfig(cmd *cobra.Command) (types2.Bacalhau, error) {
	var opts []configv2.Option
	v := viper.GetViper()
	// check if the user specified config files via the --config flag
	configFiles := getConfigFiles(v)

	// if none were provided look in $XDG_CONFIG_HOME/bacalhau/config.yaml
	if len(configFiles) == 0 {
		xdgPath, err := os.UserConfigDir()
		if err == nil {
			path := filepath.Join(xdgPath, "bacalhau", config.FileName)
			if _, err := os.Stat(path); err != nil {
				// if the file exists and could not be read, return an error
				if !os.IsNotExist(err) {
					return types2.Bacalhau{}, fmt.Errorf("loading config file at %q: %w", path, err)
				}
			} else {
				// the file exists, use it.
				configFiles = append(configFiles, path)
			}
		}
	}
	// if a config file is present, apply it to the config
	if len(configFiles) > 0 {
		opts = append(opts, configv2.WithPaths(configFiles...))
	}

	configFlags := getConfigFlags(v, cmd)
	if len(configFlags) > 0 {
		opts = append(opts, configv2.WithFlags(configFlags))
	}

	configEnvVar := getConfigEnvVars(v)
	if len(configEnvVar) > 0 {
		opts = append(opts, configv2.WithEnvironmentVariables(configEnvVar))
	}

	configValues := getConfigValues(v)
	if len(configValues) > 0 {
		opts = append(opts, configv2.WithValues(configValues))
	}

	cfg, err := configv2.New(opts...)
	if err != nil {
		return types2.Bacalhau{}, err
	}

	var out types2.Bacalhau
	if err := cfg.Unmarshal(&out); err != nil {
		return types2.Bacalhau{}, err
	}
	return out, nil
}

func getConfigValues(v *viper.Viper) map[string]any {
	return v.GetStringMap(cliflags.RootCommandConfigValues)
}

func getConfigFlags(v *viper.Viper, cmd *cobra.Command) map[string]*pflag.Flag {
	flagDefs := v.Get(cliflags.RootCommandConfigFlags)
	if flagDefs == nil {
		return nil
	}
	flagsConfigs := flagDefs.([]configflags.Definition)
	out := make(map[string]*pflag.Flag)
	for _, flag := range flagsConfigs {
		out[flag.ConfigPath] = cmd.Flags().Lookup(flag.FlagName)
	}
	return out
}

func getConfigEnvVars(v *viper.Viper) map[string][]string {
	flagDefs := v.Get(cliflags.RootCommandConfigFlags)
	if flagDefs == nil {
		return nil
	}
	flagsConfigs := flagDefs.([]configflags.Definition)
	out := make(map[string][]string)
	for _, flag := range flagsConfigs {
		out[flag.ConfigPath] = flag.EnvironmentVariables
	}
	return out
}

func getConfigFiles(v *viper.Viper) []string {
	// check if the user specified config files via the --config/-c flag
	return v.GetStringSlice(cliflags.RootCommandConfigFiles)
}
