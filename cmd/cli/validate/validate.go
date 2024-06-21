package validate

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "validate",
		Short:      "DEPRECATED: validate a job using a json or yaml file.",
		Deprecated: "This command has moved! Please use `job validate` to validate jobs.",
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
