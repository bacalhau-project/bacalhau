package cancel

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	cancelCmd := &cobra.Command{
		Use:        "cancel [id]",
		Short:      "DEPRECATED: Cancel a previously submitted job",
		Deprecated: "This command has moved! Please use `job stop` to cancel jobs.",
		Args:       cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return nil
		},
	}

	return cancelCmd
}
