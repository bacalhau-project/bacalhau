package node

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "node",
		Short:              "Commands to query and update nodes information.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewListCmd())
	return cmd
}
