package util

import (
	"fmt"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

func SetupRepoConfig() (types.BacalhauConfig, error) {
	// get the global viper instance
	v := viper.GetViper()
	// get the repo path set in the root command.
	repoPath := v.GetString("repo")
	if repoPath == "" {
		return types.BacalhauConfig{}, fmt.Errorf("repo path not set")
	}
	cfg := config.New(config.WithViper(v))
	// create or open the bacalhau repo and load the config
	_, err := setup.SetupBacalhauRepo(repoPath, cfg)
	if err != nil {
		return types.BacalhauConfig{}, fmt.Errorf("failed to reconcile repo: %w", err)
	}
	return cfg.Current()
}
