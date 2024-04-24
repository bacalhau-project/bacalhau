package job

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"

	"github.com/spf13/cobra"
)

func NewCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "job",
		Short:              "Commands to submit, query and update jobs.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	cmd.AddCommand(NewDescribeCmd(cfg))
	cmd.AddCommand(NewExecutionCmd(cfg))
	cmd.AddCommand(NewHistoryCmd(cfg))
	cmd.AddCommand(NewListCmd(cfg))
	cmd.AddCommand(NewLogCmd(cfg))
	cmd.AddCommand(NewRunCmd(cfg))
	cmd.AddCommand(NewStopCmd(cfg))
	return cmd
}
