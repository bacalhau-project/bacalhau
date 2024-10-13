package deprecated

import (
	"github.com/spf13/cobra"
)

func NewExecCommand() *cobra.Command {
	cancelCmd := &cobra.Command{
		Use:        "exec",
		Deprecated: "exec was an experimental feature and no longer supported",
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return nil
		},
	}

	return cancelCmd
}
