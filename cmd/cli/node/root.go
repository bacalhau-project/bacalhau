package node

import (
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "node",
		Short:              "Commands to query and update nodes information.",
		PersistentPreRunE:  util.AfterParentPreRunHook(util.ClientPreRunHooks),
		PersistentPostRunE: util.AfterParentPostRunHook(util.ClientPostRunHooks),
	}

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewListCmd())
	return cmd
}
