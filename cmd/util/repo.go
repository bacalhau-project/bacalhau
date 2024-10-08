package util

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

func SetupRepoConfig(cmd *cobra.Command) (types.Bacalhau, error) {
	cfg, err := SetupConfig(cmd)
	if err != nil {
		return types.Bacalhau{}, err
	}
	// create or open the bacalhau repo and load the config
	r, err := SetupRepo(cfg)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("failed to reconcile repo: %w", err)
	}

	// TODO(forrest): we need to start this hook somewhere else as not all CLI methods call this parent method.
	hook.StartUpdateCheck(cmd, cfg, r)

	return cfg, nil
}

func SetupRepo(cfg types.Bacalhau) (*repo.FsRepo, error) {
	// create or open the bacalhau repo and load the config
	r, err := setup.SetupBacalhauRepo(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile repo: %w", err)
	}
	return r, nil
}

func SetupConfigType(cmd *cobra.Command) (*config.Config, error) {
	var opts []config.Option
	v := viper.GetViper()
	// check if the user specified config files via the --config flag
	configFiles := getConfigFiles(v)

	// apply user specified config files via the --config flag, if any
	if len(configFiles) > 0 {
		opts = append(opts, config.WithPaths(getConfigFiles(v)...))
	}

	configFlags, err := getConfigFlags(v, cmd)
	if err != nil {
		return nil, err
	}
	if len(configFlags) > 0 {
		opts = append(opts, config.WithFlags(configFlags))
	}

	configEnvVar := getConfigEnvVars(v)
	if len(configEnvVar) > 0 {
		opts = append(opts, config.WithEnvironmentVariables(configEnvVar))
	}

	configValues := getConfigValues(v)
	if len(configValues) > 0 {
		opts = append(opts, config.WithValues(configValues))
	}

	cfg, err := config.New(opts...)
	if err != nil {
		return nil, err
	}

	// We always apply the configured logging level. Logging mode on the other hand is only applied with serve cmd
	if err = logger.ParseAndConfigureLoggingLevel(cfg.Get(types.LoggingLevelKey).(string)); err != nil {
		return nil, fmt.Errorf("failed to configure logging: %w", err)
	}

	return cfg, nil
}

func SetupConfig(cmd *cobra.Command) (types.Bacalhau, error) {
	cfg, err := SetupConfigType(cmd)
	if err != nil {
		return types.Bacalhau{}, err
	}
	return UnmarshalBacalhauConfig(cfg)
}

func SetupConfigs(cmd *cobra.Command) (types.Bacalhau, *config.Config, error) {
	cfg, err := SetupConfigType(cmd)
	if err != nil {
		return types.Bacalhau{}, nil, err
	}
	bacalhauCfg, err := UnmarshalBacalhauConfig(cfg)
	if err != nil {
		return types.Bacalhau{}, nil, err
	}
	return bacalhauCfg, cfg, nil
}

func UnmarshalBacalhauConfig(cfg *config.Config) (types.Bacalhau, error) {
	var out types.Bacalhau
	if err := cfg.Unmarshal(&out); err != nil {
		return types.Bacalhau{}, err
	}
	return out, nil
}

func getConfigValues(v *viper.Viper) map[string]any {
	return v.GetStringMap(cliflags.RootCommandConfigValues)
}

func getConfigFlags(v *viper.Viper, cmd *cobra.Command) (map[string][]*pflag.Flag, error) {
	flagDefs := v.Get(cliflags.RootCommandConfigFlags)
	if flagDefs == nil {
		return nil, nil
	}
	flagsConfigs := flagDefs.([]configflags.Definition)
	out := make(map[string][]*pflag.Flag)
	for _, flag := range flagsConfigs {
		out[flag.ConfigPath] = append(out[flag.ConfigPath], cmd.Flags().Lookup(flag.FlagName))
	}
	return out, nil
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
