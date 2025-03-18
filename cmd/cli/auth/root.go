package auth

import (
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "auth",
		Short:              "Authentication commands for Bacalhau",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	cmd.AddCommand(NewSSOCmd())
	cmd.AddCommand(NewHashPasswordCmd())
	return cmd
}
