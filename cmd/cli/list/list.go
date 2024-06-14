package list

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "list",
		Short:      "DEPRECATED: List jobs on the network",
		Deprecated: "This command has moved! Please use `job list` to list jobs.",
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
