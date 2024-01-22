package agent

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/spf13/cobra"
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
	return cmd
}
