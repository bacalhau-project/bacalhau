package deprecated

import (
	"github.com/spf13/cobra"
)

func NewValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "validate",
		Short:      "DEPRECATED: validate a job using a json or yaml file.",
		Deprecated: "Please use `job validate` to validate jobs.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
