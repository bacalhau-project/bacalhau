package deprecated

import (
	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:        "create",
		Short:      "DEPRECATED: Create a job using a json or yaml file.",
		Deprecated: "Please use `bacalhau job run` to create jobs.\n" + migrationMessageSuffix,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return nil
		},
	}
	return createCmd
}
