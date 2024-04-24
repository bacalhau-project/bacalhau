package node

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func NewCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "node",
		Short:              "Commands to query and update nodes information.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	cmd.AddCommand(NewDescribeCmd(cfg))
	cmd.AddCommand(NewListCmd(cfg))

	// Approve Action
	cmd.AddCommand(NewActionCmd(cfg, apimodels.NodeActionApprove))

	// Reject Action
	cmd.AddCommand(NewActionCmd(cfg, apimodels.NodeActionReject))

	// Reject Action
	cmd.AddCommand(NewActionCmd(cfg, apimodels.NodeActionDelete))

	return cmd
}
