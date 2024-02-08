package config

import (
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func newDefaultCmd() *cobra.Command {
	showCmd := &cobra.Command{
		Use:   "default",
		Short: "Show the default bacalhau config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return defaultConfig(cmd)
		},
	}
	showCmd.PersistentFlags().String("path", viper.GetString("repo"), "sets path dependent config fields")
	return showCmd
}

func defaultConfig(cmd *cobra.Command) error {
	// clear any existing configuration before generating the default.
	config.Reset()
	defaultConfig, err := config.Init(cmd.Flag("path").Value.String())
	if err != nil {
		return err
	}
	cfgbytes, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return err
	}
	cmd.Println(string(cfgbytes))
	return nil
}
