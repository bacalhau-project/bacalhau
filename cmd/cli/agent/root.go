package agent

import (
	"github.com/bacalhau-project/bacalhau/cmd/cli/agent/license"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "agent",
		Short:              "Commands to query agent information.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}
	cmd.AddCommand(NewAliveCmd())
	cmd.AddCommand(NewNodeCmd())
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewConfigCmd())
	cmd.AddCommand(license.NewAgentLicenseRootCmd())
	return cmd
}
