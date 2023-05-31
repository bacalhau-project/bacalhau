package bacalhau

import (
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:               "run",
		Short:             "Run a job on the network (see subcommands for supported flavors)",
		PreRun:            applyPorcelainLogLevel,
		PersistentPreRunE: checkVersion,
	}
	runCmd.AddCommand(newRunPythonCmd())
	return runCmd
}
