package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
)

var jobEngine string
var jobVerifier string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobEnv []string
var jobConcurrency int
var jobLabels []string
var skipSyntaxChecking bool

// For testing
var flagClearLabels bool

func init() {
	runCmd.PersistentFlags().StringVar(
		&jobEngine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	runCmd.PersistentFlags().StringVar(
		&jobVerifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
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
	runCmd.PersistentFlags().StringSliceVarP(&jobLabels,
		"labels", "l", []string{},
		`List of labels for the job. In the format 'a,b,c,1'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`,
	)

	// For testing
	runCmd.PersistentFlags().BoolVar(&flagClearLabels,
		"clear-labels", false,
		`Clear all labels before executing. For testing purposes only, should never be necessary in the real world.`,
	)
	runCmd.PersistentFlags().MarkHidden("clear-labels")
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a job on the network",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint

		jobImage := cmdArgs[0]
		jobEntrypoint := cmdArgs[1:]

		spec, deal, err := job.ConstructJob(
			jobEngine,
			jobVerifier,
			jobInputVolumes,
			jobOutputVolumes,
			jobEnv,
			jobEntrypoint,
			jobImage,
			jobConcurrency,
			jobLabels,
		)

		if err != nil {
			return err
		}

		if !skipSyntaxChecking {
			err := system.CheckBashSyntax(jobEntrypoint)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(spec, deal)
		if err != nil {
			return err
		}

		if flagClearLabels {
			clearLabels()
		}

		fmt.Printf("%s\n", job.Id)
		return nil
	},
}

func clearLabels() {
	// For testing purposes - just clear the labels before we execute
	jobLabels = []string{}
}
