package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const CompleteStatus = "Complete"
const DefaultDockerRunWaitSeconds = 100

var jobEngine string
var jobVerifier string
var jobInputs []string
var jobInputUrls []string
var jobInputVolumes []string
var jobOutputVolumes []string
var jobEnv []string
var jobConcurrency int
var jobCPU string
var jobMemory string
var jobGPU string
var jobWorkingDir string
var skipSyntaxChecking bool
var isLocal bool
var waitForJobToFinish bool
var waitForJobToFinishAndPrintOutput bool
var waitForJobTimeoutSecs int
var jobLabels []string
var shardingGlobPattern string
var shardingBasePath string
var shardingBatchSize int

var runDownloadFlags = ipfs.DownloadSettings{
	TimeoutSecs:    10,
	OutputDir:      ".",
	IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
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
		`URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz' mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz') and supports HTTP and HTTPS.`, //nolint:lll // Documentation, ok if long.
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

	dockerRunCmd.PersistentFlags().StringVarP(
		&jobWorkingDir, "workdir", "w", "",
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(&jobLabels,
		"labels", "l", []string{},
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&waitForJobToFinish, "wait", false,
		`Wait for the job to finish.`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&isLocal, "local", false,
		`Run the job locally. Docker is required`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&waitForJobTimeoutSecs, "wait-timeout-secs", DefaultDockerRunWaitSeconds,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&waitForJobToFinishAndPrintOutput, "download", false,
		`Download the results and print stdout once the job has completed (implies --wait).`,
	)

	setupDownloadFlags(dockerRunCmd, &runDownloadFlags)

	dockerRunCmd.PersistentFlags().StringVar(
		&shardingGlobPattern, "sharding-glob-pattern", "",
		`Use this pattern to match files to be sharded.`,
	)

	dockerRunCmd.PersistentFlags().StringVar(
		&shardingBasePath, "sharding-base-path", "",
		`Where the sharding glob pattern starts from - useful when you have multiple volumes.`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&shardingBatchSize, "sharding-batch-size", 1,
		`Place results of the sharding glob pattern into groups of this size.`,
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
		isLocal = false
		skipSyntaxChecking = false
		waitForJobToFinish = false
		waitForJobToFinishAndPrintOutput = false
		runDownloadFlags = ipfs.DownloadSettings{
			TimeoutSecs:    10,
			OutputDir:      ".",
			IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
		}
		jobWorkingDir = ""
		shardingGlobPattern = ""
		shardingBasePath = ""
		shardingBatchSize = 1
	},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := context.Background()
		jobImage := cmdArgs[0]
		jobEntrypoint := cmdArgs[1:]

		if waitForJobToFinishAndPrintOutput {
			waitForJobToFinish = true
		}

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

		if len(jobWorkingDir) > 0 {
			err = system.ValidateWorkingDir(jobWorkingDir)
			if err != nil {
				return err
			}
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
			jobWorkingDir,
		)

		if err != nil {
			return err
		}

		spec.Sharding = executor.JobShardingConfig{
			GlobPattern: shardingGlobPattern,
			BasePath:    shardingBasePath,
			BatchSize:   shardingBatchSize,
		}

		if !skipSyntaxChecking {
			err = system.CheckBashSyntax(jobEntrypoint)
			if err != nil {
				return err
			}
		}
		if isLocal {
			fmt.Printf("LOCAL --------------------------------------\n")
			spew.Dump("LOCAL")
		} else {
			job, err := getAPIClient().Submit(ctx, spec, deal, nil)
			if err != nil {
				return err
			}

			cmd.Printf("%s\n", job.ID)
			if waitForJobToFinish {
				resolver := getAPIClient().GetJobStateResolver()
				resolver.SetWaitTime(waitForJobTimeoutSecs, time.Second*1)
				err = resolver.WaitUntilComplete(ctx, job.ID)
				if err != nil {
					return err
				}

				if waitForJobToFinishAndPrintOutput {
					results, err := getAPIClient().GetResults(ctx, job.ID)
					if err != nil {
						return err
					}
					if len(results) == 0 {
						return fmt.Errorf("no results found")
					}
					err = ipfs.DownloadJob(
						cm,
						job,
						results,
						runDownloadFlags,
					)
					if err != nil {
						return err
					}
					body, err := os.ReadFile(filepath.Join(runDownloadFlags.OutputDir, "stdout"))
					if err != nil {
						return err
					}
					fmt.Println()
					fmt.Println(string(body))
				}
			}
		}

		return nil
	},
}
