package profile

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:      "profile",
		Short:    "Manage CLI connection profiles for Bacalhau clusters.",
		PreRunE:  hook.ClientPreRunHooks,
		PostRunE: hook.ClientPostRunHooks,
	}
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewSaveCmd())
	cmd.AddCommand(NewShowCmd())
	cmd.AddCommand(NewSelectCmd())
	cmd.AddCommand(NewDeleteCmd())
	return cmd
}
