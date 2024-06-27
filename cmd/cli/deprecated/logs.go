package deprecated

import (
	"github.com/spf13/cobra"
)

func NewLogsCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "logs [id]",
		Short:      "DEPRECATED: Follow logs from a currently executing job",
		Deprecated: "Please use `job logs` to follow job logs.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
