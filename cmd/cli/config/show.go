package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
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
		c := config.New()
		currentConfig, err = c.Init("")
		if err != nil {
			return err
		}
	} else {
		currentConfig, err = cfg.Init(repoPath)
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
