package node

import "github.com/spf13/cobra"

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Commands to query and update nodes information.",
	}

	cmd.AddCommand(NewDescribeCmd())
	cmd.AddCommand(NewListCmd())
	return cmd
}
