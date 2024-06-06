package config

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

func newDefaultCmd() *cobra.Command {
	defaultCmd := &cobra.Command{
		Use:  "print",
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return defaultConfig(cmd)
		},
	}
	return defaultCmd
}

func defaultConfig(cmd *cobra.Command) error {
	c := config.New()
	def, err := c.Current()
	if err != nil {
		return err
	}
	str, err := yaml.Marshal(def)
	if err != nil {
		return err
	}
	cmd.Println(string(str))
	return nil
}
