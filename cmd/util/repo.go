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
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

func SetupRepoConfig(cmd *cobra.Command) (types.BacalhauConfig, error) {
	cfg, err := SetupConfig(cmd)
	if err != nil {
		return types.BacalhauConfig{}, err
	}
	// create or open the bacalhau repo and load the config
	r, err := SetupRepo(cfg)
	if err != nil {
		return types.BacalhauConfig{}, fmt.Errorf("failed to reconcile repo: %w", err)
	}

	bacalhauCfg, err := cfg.Current()
	if err != nil {
		return types.BacalhauConfig{}, fmt.Errorf("failed to load config: %w", err)
	}

	hook.StartUpdateCheck(cmd, bacalhauCfg, r)

	return bacalhauCfg, nil
}

func SetupRepo(cfg config.ReadWriter) (*repo.FsRepo, error) {
	// get the global viper instance
	v := viper.GetViper()
	// get the repo path set in the root command, and ensure it's set (it will always be set unless dev error)
	repoPath := v.GetString("repo")
	if repoPath == "" {
		return nil, fmt.Errorf("repo path not set")
	}
	// create or open the bacalhau repo and load the config
	r, err := setup.SetupBacalhauRepo(repoPath, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to reconcile repo: %w", err)
	}
	return r, nil
}

func SetupConfig(cmd *cobra.Command) (config.ReadWriter, error) {
	// get the global viper instance
	v := viper.GetViper()

	// get the repo path set in the root command, and ensure it's set (it will always be set unless dev error)
	repoPath := v.GetString("repo")
	if repoPath == "" {
		return nil, fmt.Errorf("repo path not set")
	}

	opts := []config.Option{config.WithDefault(config.ForEnvironment())}

	// check if the user specified config files via the --config flag
	configFiles, err := getConfigFiles(v, repoPath)
	if err != nil {
		return nil, err
	}
	// if a config file is present, apply it to the config
	if len(configFiles) > 0 {
		opts = append(opts, config.WithPaths(configFiles...))
	}

	configFlags := getConfigFlags(v, cmd)
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

	return cfg, nil
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

func getConfigFiles(v *viper.Viper, repoPath string) ([]string, error) {
	// check if the user specified config files via the --config/-c flag
	configFiles := v.GetStringSlice(cliflags.RootCommandConfigFiles)
	if len(configFiles) > 0 {
		return configFiles, nil
	}

	// check if a config file is present at $XDG_CONFIG_HOME/bacalhau/config.yaml
	{
		xdgPath, err := os.UserConfigDir()
		if err == nil {
			path := filepath.Join(xdgPath, config.FileName)
			if _, err := os.Stat(path); err != nil {
				// if the file exists and could not be read, return an error
				if !os.IsNotExist(err) {
					return nil, fmt.Errorf("loading xdg config file at %q: %w", path, err)
				}
			} else {
				// the file exists, use it.
				return []string{path}, nil
			}
		}
	}

	// if no config files were provided by the users, and we failed to find a config in
	// $XDG_CONFIG_HOME/bacalhau/config.yaml fall back to the repo
	{
		path := filepath.Join(repoPath, config.FileName)
		if _, err := os.Stat(path); err != nil {
			// if the file exists and could not be read, return an error
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading xdg config file at %q: %w", path, err)
			}
		} else {
			// the file exists, use it.
			return []string{path}, nil
		}
	}

	// no config file exists, this is fine bacalhau will use defaults and values from flags and env vars.
	return []string{}, nil
}
