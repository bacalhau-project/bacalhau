package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const CompleteStatus = "Complete"

var jobEngine string
var jobVerifier string
var jobInputs []string
var jobInputUrls []string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobLocalOutput string
var jobEnv []string
var jobConcurrency int
var jobCPU string
var jobMemory string
var jobGPU string
var skipSyntaxChecking bool
var waitForJobToFinishAndPrintOutput bool
var jobLabels []string

var runDownloadFlags = downloadSettings{
	timeoutSecs:    10,
	outputDir:      ".",
	ipfsSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
}

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
		&jobInputs, "inputs", "i", []string{},
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputUrls, "input-urls", "u", []string{},
		`URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz' mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz') and supports HTTP and HTTPS.`, // nolint:lll // Documentation, ok if long.
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobInputVolumes, "input-volumes", "v", []string{},
		`CID:path of the input data volumes, if you need to set the path of the mounted data.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&jobOutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes. 'outputs:/outputs' is always added.`,
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
	dockerRunCmd.PersistentFlags().StringVar(
		&jobGPU, "gpu", "",
		`Job GPU requirement (e.g. 1, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().BoolVar(
		&skipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(&jobLabels,
		"labels", "l", []string{},
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, // nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVarP(
		&waitForJobToFinishAndPrintOutput, "wait", "w", false,
		`Wait For Job To Finish And Print Output`,
	)

	setupDownloadFlags(dockerRunCmd, runDownloadFlags)
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Run a docker job on the network (see run subcommand)",
}

var dockerRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a docker job on the network",
	Args:  cobra.MinimumNArgs(1),
	PostRun: func(cmd *cobra.Command, args []string) {
		// Can't think of any reason we'd want these to persist.
		// The below is to clean out for testing purposes. (Kinda ugly to put it in here,
		// but potentially cleaner than making things public, which would
		// be the other way to attack this.)
		jobInputs = []string{}
		jobInputUrls = []string{}
		jobInputVolumes = []string{}
		jobOutputVolumes = []string{}
		jobEnv = []string{}
		jobLabels = []string{}

		jobEngine = "docker"
		jobVerifier = "ipfs"
		jobConcurrency = 1
		jobCPU = ""
		jobMemory = ""
		jobGPU = ""
		skipSyntaxChecking = false
		waitForJobToFinishAndPrintOutput = false
		runDownloadFlags = downloadSettings{
			timeoutSecs:    10,
			outputDir:      ".",
			ipfsSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
		}
	},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
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

		for _, i := range jobInputs {
			jobInputVolumes = append(jobInputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		jobOutputVolumes = append(jobOutputVolumes, "outputs:/outputs")

		// No error checking, because it will never be an error (for now)
		sanitizationMsgs, sanitizationFatal := system.SanitizeImageAndEntrypoint(jobEntrypoint)
		if sanitizationFatal {
			log.Error().Msgf("Errors: %+v", sanitizationMsgs)
			return fmt.Errorf("could not continue with errors")
		}

		if len(sanitizationMsgs) > 0 {
			log.Warn().Msgf("Found the following possible errors in arguments: %+v", sanitizationMsgs)
		}

		spec, deal, err := jobutils.ConstructDockerJob(
			engineType,
			verifierType,
			jobCPU,
			jobMemory,
			jobGPU,
			jobInputUrls,
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
		if waitForJobToFinishAndPrintOutput {
			resolver, err := getAPIClient().GetJobStateResolver(ctx, job.ID)
			if err != nil {
				return err
			}
			err = resolver.WaitUntilComplete(ctx)
			if err != nil {
				return err
			}
			resultCIDs, err := resolver.GetResults()
			if err != nil {
				return err
			}
			if len(resultCIDs) == 0 {
				return fmt.Errorf("no result CIDs found")
			}
			err = downloadJobResults(
				cm,
				[]string{resultCIDs[0]},
				runDownloadFlags,
			)
			if err != nil {
				return err
			}
			body, err := os.ReadFile(filepath.Join(runDownloadFlags.outputDir, resultCIDs[0], "stdout"))
			if err != nil {
				return err
			}
			fmt.Println()
			fmt.Println(string(body))
		}

		return nil
	},
}
