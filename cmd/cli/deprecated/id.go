package deprecated

import (
	"github.com/spf13/cobra"
)

func NewIDCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "id",
		Short:      "DEPRECATED: Show bacalhau node id info",
		Deprecated: "Please use `bacalhau agent node` to inspect bacalhau nodes.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, _ []string) error { return nil },
	}
}
