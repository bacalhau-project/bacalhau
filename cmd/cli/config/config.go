package config

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:      "config",
		Short:    "Interact with the bacalhau configuration system.",
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
	}
	configCmd.AddCommand(newListCmd())
	configCmd.AddCommand(newSetCmd())
	configCmd.AddCommand(newAutoResourceCmd())
	return configCmd
}
