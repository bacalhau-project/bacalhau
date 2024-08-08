package util

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

func SetupRepoConfig(cmd *cobra.Command) (types.BacalhauConfig, error) {
	cfg, err := SetupConfig()
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

func SetupConfig() (config.ReadWriter, error) {
	// get the global viper instance
	v := viper.GetViper()
	// get the repo path set in the root command, and ensure it's set (it will always be set unless dev error)
	repoPath := v.GetString("repo")
	if repoPath == "" {
		return nil, fmt.Errorf("repo path not set")
	}
	// check if the user specified config files via the --config flag
	configFiles := v.GetStringSlice(cliflags.RootCommandConfigFiles)
	if len(configFiles) == 0 {
		// if no files were provided we look in repo and xdg config, the latter takes precedence
		repoConfig, err := getConfigFileAtPath(repoPath)
		if err != nil {
			return nil, err
		}
		var xdgConfig string
		xdgPath, err := os.UserConfigDir()
		if err != nil {
			log.Info().Err(err).Msg("find to find user config dir")
		} else {
			xdgConfig, err = getConfigFileAtPath(xdgPath)
			if err != nil {
				return nil, err
			}
		}
		configFiles = append(configFiles, repoConfig, xdgConfig)
	}

	// merge values provided via env vars and flags with values provided via --config, the latter takes precedence
	configValues := mergeConfigValuesWithFlags(v)

	cfg, err := config.New(
		config.WithValues(configValues),
		config.WithPaths(configFiles...),
	)
	if err != nil {
		return nil, err
	}

	return cfg, nil
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

func mergeConfigValuesWithFlags(v *viper.Viper) map[string]any {
	base := v.AllSettings()
	override := v.GetStringMap(cliflags.RootCommandConfigValues)
	for k, v := range override {
		base[k] = v
	}
	return base
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
