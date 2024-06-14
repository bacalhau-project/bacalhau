package create

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	createCmd := &cobra.Command{
		Use:        "create",
		Short:      "DEPRECATED: Create a job using a json or yaml file.",
		Deprecated: "This command has moved! Please use `job run` to create jobs.",
		Args:       cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return nil
		},
	}
	return createCmd
}
