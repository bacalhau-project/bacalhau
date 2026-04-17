package job

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "job",
		Short:              "Commands to submit, query and update jobs.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	// Register profile flag for client commands
	cliflags.RegisterProfileFlag(cmd)

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewExecutionCmd())
	cmd.AddCommand(NewHistoryCmd())
	cmd.AddCommand(NewVersionsCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewLogCmd())
	cmd.AddCommand(NewRunCmd())
	cmd.AddCommand(NewRerunCmd())
	cmd.AddCommand(NewStopCmd())
	cmd.AddCommand(NewGetCmd())
	cmd.AddCommand(NewValidateCmd())
	return cmd
}
