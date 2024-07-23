package deprecated

import (
	"github.com/spf13/cobra"
)

func NewCancelCmd() *cobra.Command {
	cancelCmd := &cobra.Command{
		Use:        "cancel [id]",
		Short:      "DEPRECATED: Cancel a previously submitted job",
		Deprecated: "Please use `bacalhau job stop` to cancel jobs.\n" + migrationMessageSuffix,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return nil
		},
	}

	return cancelCmd
}
