package deprecated

import (
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "list",
		Short:      "DEPRECATED: List jobs on the network",
		Deprecated: "Please use `bacalhau job list` to list jobs.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
