package id

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "id",
		Short:      "DEPRECATED: Show bacalhau node id info",
		Deprecated: "This command has moved! Please use `agent node` to inspect bacalhau nodes.",
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
