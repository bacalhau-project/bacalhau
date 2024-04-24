package config

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

func newShowCmd() *cobra.Command {
	showCmd := &cobra.Command{
		Use:   "show",
		Short: "Show the current bacalhau config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showConfig(cmd)
		},
	}
	showCmd.PersistentFlags().String("path", viper.GetString("repo"), "sets path dependent config fields")
	return showCmd
}

func showConfig(cmd *cobra.Command) error {
	// clear any existing configuration before generating the current.

	c := config.New(config.ForEnvironment())
	currentConfig, err := c.Init(cmd.Flag("path").Value.String())
	if err != nil {
		return err
	}
	cfgbytes, err := yaml.Marshal(currentConfig)
	if err != nil {
		return err
	}
	cmd.Println(string(cfgbytes))
	return nil
}
