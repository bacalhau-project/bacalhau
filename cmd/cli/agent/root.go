package agent

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/config"

	"github.com/spf13/cobra"
)

func NewCmd(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "agent",
		Short:              "Commands to query agent information.",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}
	cmd.AddCommand(NewAliveCmd(cfg))
	cmd.AddCommand(NewNodeCmd(cfg))
	cmd.AddCommand(NewVersionCmd(cfg))
	return cmd
}
