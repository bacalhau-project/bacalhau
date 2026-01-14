package auth

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/cli/auth/sso"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "auth",
		Short:              "Authentication commands for Bacalhau",
		PersistentPreRunE:  hook.AfterParentPreRunHook(hook.RemoteCmdPreRunHooks),
		PersistentPostRunE: hook.AfterParentPostRunHook(hook.RemoteCmdPostRunHooks),
	}

	// Register profile flag for client commands
	cliflags.RegisterProfileFlag(cmd)

	cmd.AddCommand(sso.NewSSORootCmd())
	cmd.AddCommand(NewHashPasswordCmd())
	cmd.AddCommand(NewInfoCmd())
	return cmd
}
