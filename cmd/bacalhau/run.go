package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/spf13/cobra"
)

var jobEngine string
var jobCids []string
var jobEnv []string
var jobEntrypoint []string
var jobImage string
var jobConcurrency int
var skipSyntaxChecking bool

func init() {
	runCmd.PersistentFlags().StringVar(
		&jobEngine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	runCmd.PersistentFlags().StringSliceVar(
		&jobCids, "cids", []string{},
		`The cids of the data used by the job (comma separated, or specify multiple times)`,
	)
	runCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	runCmd.PersistentFlags().IntVar(
		&jobConcurrency, "concurrency", 1,
		`How many nodes should run the job`,
	)
	runCmd.PersistentFlags().StringVar(
		&jobImage, "image", "ubuntu:latest",
		`What image do we use for the job`,
	)
	runCmd.PersistentFlags().StringSliceVar(
		&jobEntrypoint, "entrypoint", []string{},
		`The entrypoint to use for the container`,
	)
	runCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint
		_, err := job.RunJob(
			jobEngine,
			jobCids,
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
