package agent

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Commands to query agent information.",
	}

	cmd.AddCommand(NewAliveCmd())
	cmd.AddCommand(NewNodeCmd())
	cmd.AddCommand(NewVersionCmd())
	return cmd
}
