package config

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"

	"github.com/spf13/cobra"
)

func NewCmd(cfg *config.Config) *cobra.Command {
	configCmd := &cobra.Command{
		Use:      "config",
		Short:    "Interact with the bacalhau configuration system.",
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
	}
	configCmd.AddCommand(newShowCmd(cfg))
	configCmd.AddCommand(newDefaultCmd())
	configCmd.AddCommand(newListCmd(cfg))
	configCmd.AddCommand(newSetCmd(cfg))
	configCmd.AddCommand(newAutoResourceCmd(cfg))
	return configCmd
}
