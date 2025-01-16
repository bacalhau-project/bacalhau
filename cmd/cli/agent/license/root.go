package license

import (
	"github.com/spf13/cobra"
)

func NewAgentLicenseRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Commands to interact with the orchestrator license",
	}

	cmd.AddCommand(NewAgentLicenseInspectCmd())
	return cmd
}
