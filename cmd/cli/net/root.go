package net

import (
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "net",
		Short:              "Manage P2P Network.",
		PersistentPreRunE:  util.AfterParentPreRunHook(util.ClientPreRunHooks),
		PersistentPostRunE: util.AfterParentPostRunHook(util.ClientPostRunHooks),
	}

	cmd.AddCommand(NewPeersCmd())
	cmd.AddCommand(NewConnectCmd())
	cmd.AddCommand(NewDisconnectCmd())
	cmd.AddCommand(NewPingCmd())
	return cmd
}
