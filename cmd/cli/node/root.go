package node

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "node",
		Short:              "Commands to query and update nodes information.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	// Register profile flag for client commands
	cliflags.RegisterProfileFlag(cmd)

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewListCmd())

	// Approve Action
	cmd.AddCommand(NewActionCmd(apimodels.NodeActionApprove))

	// Reject Action
	cmd.AddCommand(NewActionCmd(apimodels.NodeActionReject))

	// Reject Action
	cmd.AddCommand(NewActionCmd(apimodels.NodeActionDelete))

	return cmd
}
