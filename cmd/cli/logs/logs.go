package logs

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "logs [id]",
		Short:      "DEPRECATED: Follow logs from a currently executing job",
		Deprecated: "This command has moved! Please use `job logs` to follow job logs.",
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
