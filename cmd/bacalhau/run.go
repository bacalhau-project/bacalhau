package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/spf13/cobra"
)

var jobEngine string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobEnv []string
var jobConcurrency int
var skipSyntaxChecking bool

func init() {
	runCmd.PersistentFlags().StringVar(
		&jobEngine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	runCmd.PersistentFlags().StringSliceVarP(
		&jobInputVolumes, "input-volumes", "v", []string{},
		`cid:path of the input data volumes`,
	)
	runCmd.PersistentFlags().StringSliceVarP(
		&jobOutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes`,
	)
	runCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	runCmd.PersistentFlags().IntVarP(
		&jobConcurrency, "concurrency", "c", 1,
		`How many nodes should run the job`,
	)
	runCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint

		jobImage := cmdArgs[0]
		jobEntrypoint := cmdArgs[1:]

		_, err := job.RunJob(
			jobEngine,
			jobInputVolumes,
			jobOutputVolumes,
			jobEnv,
			jobEntrypoint,
			jobImage,
			jobConcurrency,
			jsonrpcHost,
			jsonrpcPort,
			skipSyntaxChecking,
		)
		return err
	},
}
