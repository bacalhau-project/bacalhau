package bacalhau

import (
	"github.com/spf13/cobra"
)

// var jobEngine string
// var jobVerifier string
// var jobInputVolumes []string
// var jobOutputVolumes []string
// var jobEnv []string
// var jobConcurrency int
// var skipSyntaxChecking bool

//nolint:gochecknoinits
func init() {
	runCmd.AddCommand(runPythonCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network (see subcommands for supported flavors)",
}
