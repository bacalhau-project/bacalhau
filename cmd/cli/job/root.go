package job

import (
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "job",
		Short:              "Commands to submit, query and update jobs.",
		PersistentPreRunE:  util.AfterParentPreRunHook(util.ClientPreRunHooks),
		PersistentPostRunE: util.AfterParentPostRunHook(util.ClientPostRunHooks),
	}

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewExecutionCmd())
	cmd.AddCommand(NewHistoryCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewLogCmd())
	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewStopCmd())
	return cmd
}
