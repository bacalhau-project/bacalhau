package bacalhau

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var jobEngine string
var jobVerifier string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobEnv []string
var jobConcurrency int
var jobCPU string
var jobMemory string
var skipSyntaxChecking bool
var jobLabels []string

var flagClearLabels bool

func init() { // nolint:gochecknoinits // Using init in cobra command is idomatic
	dockerCmd.AddCommand(dockerRunCmd)

	// TODO: don't make jobEngine specifiable in the docker subcommand
	dockerRunCmd.PersistentFlags().StringVar(
		&jobEngine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobVerifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputVolumes, "input-volumes", "v", []string{},
		`cid:path of the input data volumes`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobOutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobEnv, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	dockerRunCmd.PersistentFlags().IntVarP(
		&jobConcurrency, "concurrency", "c", 1,
		`How many nodes should run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobCPU, "cpu", "",
		`Job CPU cores (e.g. 500m, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&jobMemory, "memory", "",
		`Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	dockerRunCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(&flagClearLabels,
		"clear-labels", false,
		`Clear all labels before executing. For testing purposes only, should never be necessary in the real world.`,
	)
	if err := dockerRunCmd.PersistentFlags().MarkHidden("clear-labels"); err != nil {
		log.Debug().Msgf("error hiding test flags: %v", err)
	}

	dockerRunCmd.PersistentFlags().StringSliceVarP(&jobLabels,
		"labels", "l", []string{},
		`List of labels for the job. In the format 'a,b,c,1'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`,
	)
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Run a docker job on the network (see run subcommand)",
}

var dockerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a docker job on the network",
	Args:  cobra.MinimumNArgs(1),
	PreRun: func(cmd *cobra.Command, args []string) {
		// Can't think of any reason we'd want these to persist.
		// The below is to clean out for testing purposes. (Kinda ugly to put it in here,
		// but potentially cleaner than making dockerRun or jobsLabel public, which would
		// be the other way to attack this.)
		jobLabels = []string{}
	},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		jobLabels = []string{}

		ctx := context.Background()
		jobImage := cmdArgs[0]
		jobEntrypoint := cmdArgs[1:]

		engineType, err := executor.ParseEngineType(jobEngine)
		if err != nil {
			return err
		}

		verifierType, err := verifier.ParseVerifierType(jobVerifier)
		if err != nil {
			return err
		}

		// No error checking, because it will never be an error (for now)
		_ = system.SanitizeImageAndEntrypoint(jobEntrypoint)

		spec, deal, err := job.ConstructDockerJob(
			engineType,
			verifierType,
			jobCPU,
			jobMemory,
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
			err = system.CheckBashSyntax(jobEntrypoint)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(ctx, spec, deal, nil)
		if err != nil {
			return err
		}

		cmd.Printf("%s\n", job.ID)
		return nil
	},
}
