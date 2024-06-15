package get

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "get [id]",
		Short:      "DEPRECATED: Get the results of a job",
		Deprecated: "This command has moved! Please use `job get` to download results of a job.",
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
