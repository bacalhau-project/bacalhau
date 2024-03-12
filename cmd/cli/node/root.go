package node

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
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

	// Approve Action
	cmd.AddCommand(NewActionCmd(apimodels.NodeActionApprove))

	// Reject Action
	cmd.AddCommand(NewActionCmd(apimodels.NodeActionReject))

	return cmd
}
