package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
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

	// check if the user specified config files via the --config flag
	configFiles, err := getConfigFiles(v, repoPath)
	if err != nil {
		return nil, err
	}
	configFlags := getConfigFlags(v, cmd)
	configEnvVar := getConfigEnvVars(v)
	configValues := getConfigValues(v)

	cfg, err := config.New(
		config.WithValues(configValues),
		config.WithFlags(configFlags),
		config.WithEnvironmentVariables(configEnvVar),
		config.WithPaths(configFiles...),
		config.WithDefault(config.ForEnvironment()),
	)
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
	flagsConfigs := flagDefs.([]configflags.Definition)
	out := make(map[string]*pflag.Flag)
	for _, flag := range flagsConfigs {
		out[flag.ConfigPath] = cmd.Flags().Lookup(flag.FlagName)
	}
	return out
}

func getConfigEnvVars(v *viper.Viper) map[string][]string {
	flagDefs := v.Get(cliflags.RootCommandConfigFlags)
	flagsConfigs := flagDefs.([]configflags.Definition)
	out := make(map[string][]string)
	for _, flag := range flagsConfigs {
		out[flag.ConfigPath] = flag.EnvironmentVariables
	}
	return out
}

func getConfigFiles(v *viper.Viper, repoPath string) ([]string, error) {
	// check if the user specified config files via the --config flag
	configFiles := v.GetStringSlice(cliflags.RootCommandConfigFiles)
	if len(configFiles) > 0 {
		return configFiles, nil
	}
	out := make([]string, 0)
	// if no files were provided we look in repo and xdg config, the latter takes precedence
	repoConfig, err := getConfigFileAtPath(repoPath)
	if err != nil {
		return nil, fmt.Errorf("loading repo config file at path %q: %w", repoPath, err)
	}
	if repoConfig != "" {
		out = append(out, repoConfig)
	}
	var xdgConfig string
	xdgPath, err := os.UserConfigDir()
	if err != nil {
		log.Info().Err(err).Msg("find to find user config dir")
	} else {
		xdgConfig, err = getConfigFileAtPath(xdgPath)
		if err != nil {
			return nil, fmt.Errorf("loading xdg config file at path %q: %w", xdgPath, err)
		}
		if xdgConfig != "" {
			out = append(out, xdgConfig)
		}
	}
	return out, nil
}

func getConfigFileAtPath(path string) (string, error) {
	configPath := filepath.Join(path, config.FileName)
	if _, err := os.Stat(configPath); err != nil {
		// it's okay for the file to not exist.
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return configPath, nil
}
