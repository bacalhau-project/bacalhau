package bacalhau

import (
	"github.com/spf13/cobra"
)

//nolint:gochecknoinits
func init() {
	runCmd.AddCommand(runPythonCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network (see subcommands for supported flavors)",
}
