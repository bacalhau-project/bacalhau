package config

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func newShowCmd(cfg *config.Config) *cobra.Command {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current bacalhau config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig(cmd, cfg)
		},
	}
	showCmd.PersistentFlags().String("path", viper.GetString("repo"), "sets path dependent config fields")
	return showCmd
}

func showConfig(cmd *cobra.Command, cfg *config.Config) error {
	var currentConfig types.BacalhauConfig
	if repoPath, err := cfg.RepoPath(); err != nil {
		cmd.Println("no config file present, showing default config")
		currentConfig, err = config.New().Current()
		if err != nil {
			return err
		}
	} else {
		err = cfg.Load(filepath.Join(repoPath, repo.ConfigFileName))
		if err != nil {
			return err
		}
		currentConfig, err = cfg.Current()
		if err != nil {
			return err
		}
	}

	cfgbytes, err := yaml.Marshal(currentConfig)
	if err != nil {
		return err
	}
	cmd.Println(string(cfgbytes))
	return nil
}
