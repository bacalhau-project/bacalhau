package describe

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "describe [id]",
		Short:      "DEPRECATED: Describe a job on the network",
		Deprecated: "This command has moved! Please use `job describe` to describe jobs.",
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
