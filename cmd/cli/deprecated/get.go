package deprecated

import (
	"github.com/spf13/cobra"
)

func NewGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:        "get [id]",
		Short:      "DEPRECATED: Get the results of a job",
		Deprecated: "Please use `job get` to download results of a job.\n" + migrationMessageSuffix,
		RunE:       func(cmd *cobra.Command, cmdArgs []string) error { return nil },
	}
}
