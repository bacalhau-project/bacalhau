package deprecated

import (
	"github.com/spf13/cobra"
)

func NewDescribeCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "describe [id]",
		Short:      "DEPRECATED: Describe a job on the network",
		Deprecated: "Please use `job describe` to describe jobs.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
