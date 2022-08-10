package bacalhau

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/version"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

const CompleteStatus = "Complete"
const DefaultDockerRunWaitSeconds = 100

var (
	dockerRunLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	dockerRunExample = templates.Examples(i18n.T(`
		# Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /output_images in the container.
		# All flags after the '--' are passed directly into the container for exacution.
		bacalhau docker run \
		-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
		-o results:/output_images \
		dpokidov/imagemagick \
		-- magick mogrify -resize 100x100 -quality 100 -path /output_images /input_images/*.jpg`))

	ODR = &DockerRunOptions{}
)

// DockerRunOptions declares the arguments accepted by the `docker run` command
type DockerRunOptions struct {
	Engine        string   // Executor - executor.Executor
	Verifier      string   // Verifier - verifier.Verifier
	Inputs        []string // Array of input CIDs
	InputUrls     []string // Array of input URLs (will be copied to IPFS)
	InputVolumes  []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes []string // Array of output volumes in 'name:mount point' form
	Env           []string // Array of environment variables
	Concurrency   int      // Number of concurrent jobs to run
	CPU           string
	Memory        string
	GPU           string
	WorkingDir    string   // Working directory for docker
	Labels        []string // Labels for the job on the Bacalhau network (for searching)

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking               bool // Verify the syntax using shellcheck
	WaitForJobToFinish               bool // Wait for the job to execute before exiting
	WaitForJobToFinishAndPrintOutput bool // Wait for the job to execute, and print the results before exiting
	WaitForJobTimeoutSecs            int  // Job time out in seconds

	ShardingGlobPattern string
	ShardingBasePath    string
	ShardingBatchSize   int
}

var runDownloadFlags = ipfs.DownloadSettings{
	TimeoutSecs:    10,
	OutputDir:      ".",
	IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	dockerCmd.AddCommand(dockerRunCmd)

	// TODO: don't make jobEngine specifiable in the docker subcommand
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Engine, "engine", "docker",
		`What executor engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Verifier, "verifier", "ipfs",
		`What verification engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Inputs, "inputs", "i", []string{},
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	//nolint:lll // Documentation, ok if long.
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.InputUrls, "input-urls", "u", []string{},
		`URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz'
		mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz')
		and supports HTTP and HTTPS.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.InputVolumes, "input-volumes", "v", []string{},
		`CID:path of the input data volumes, if you need to set the path of the mounted data.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.OutputVolumes, "output-volumes", "o", []string{},
		`name:path of the output data volumes. 'outputs:/outputs' is always added.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Env, "env", "e", []string{},
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	dockerRunCmd.PersistentFlags().IntVarP(
		&ODR.Concurrency, "concurrency", "c", 1,
		`How many nodes should run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.CPU, "cpu", "",
		`Job CPU cores (e.g. 500m, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Memory, "memory", "",
		`Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.GPU, "gpu", "",
		`Job GPU requirement (e.g. 1, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.SkipSyntaxChecking, "skip-syntax-checking", false,
		`Skip having 'shellchecker' verify syntax of the command`,
	)

	dockerRunCmd.PersistentFlags().StringVarP(
		&ODR.WorkingDir, "workdir", "w", "",
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Labels, "labels", "l", []string{},
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.WaitForJobToFinish, "wait", false,
		`Wait for the job to finish.`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.WaitForJobToFinishAndPrintOutput, "download", false,
		`Download the results and print stdout once the job has completed (implies --wait).`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&ODR.WaitForJobTimeoutSecs, "wait-timeout-secs", DefaultDockerRunWaitSeconds,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`,
	)

	setupDownloadFlags(dockerRunCmd, &runDownloadFlags)

	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.ShardingGlobPattern, "sharding-glob-pattern", "",
		`Use this pattern to match files to be sharded.`,
	)

	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.ShardingBasePath, "sharding-base-path", "",
		`Where the sharding glob pattern starts from - useful when you have multiple volumes.`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&ODR.ShardingBatchSize, "sharding-batch-size", 1,
		`Place results of the sharding glob pattern into groups of this size.`,
	)
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Run a docker job on the network (see run subcommand)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that the server version is compatible with the client version
		serverVersion, _ := getAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			log.Err(err)
			return err
		}
		return nil
	},
}

var dockerRunCmd = &cobra.Command{
	Use:     "run",
	Short:   "Run a docker job on the network",
	Long:    dockerRunLong,
	Example: dockerRunExample,
	Args:    cobra.MinimumNArgs(1),
	PostRun: func(cmd *cobra.Command, args []string) {
	},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := context.Background()
		ODR.Image = cmdArgs[0]
		ODR.Entrypoint = cmdArgs[1:]

		if ODR.WaitForJobToFinishAndPrintOutput {
			ODR.WaitForJobToFinish = true
		}

		engineType, err := executor.ParseEngineType(ODR.Engine)
		if err != nil {
			return err
		}

		verifierType, err := verifier.ParseVerifierType(ODR.Verifier)
		if err != nil {
			return err
		}

		for _, i := range ODR.Inputs {
			ODR.InputVolumes = append(ODR.InputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		// No error checking, because it will never be an error (for now)
		sanitizationMsgs, sanitizationFatal := system.SanitizeImageAndEntrypoint(ODR.Entrypoint)
		if sanitizationFatal {
			log.Error().Msgf("Errors: %+v", sanitizationMsgs)
			return fmt.Errorf("could not continue with errors")
		}

		if len(sanitizationMsgs) > 0 {
			log.Warn().Msgf("Found the following possible errors in arguments: %+v", sanitizationMsgs)
		}

		if len(ODR.WorkingDir) > 0 {
			err = system.ValidateWorkingDir(ODR.WorkingDir)
			if err != nil {
				return err
			}
		}

		spec, deal, err := jobutils.ConstructDockerJob(
			engineType,
			verifierType,
			ODR.CPU,
			ODR.Memory,
			ODR.GPU,
			ODR.InputUrls,
			ODR.InputVolumes,
			ODR.OutputVolumes,
			ODR.Env,
			ODR.Entrypoint,
			ODR.Image,
			ODR.Concurrency,
			ODR.Labels,
			ODR.WorkingDir,
			doNotTrack,
		)

		if err != nil {
			return err
		}

		spec.Sharding = executor.JobShardingConfig{
			GlobPattern: ODR.ShardingGlobPattern,
			BasePath:    ODR.ShardingBasePath,
			BatchSize:   ODR.ShardingBatchSize,
		}

		if !ODR.SkipSyntaxChecking {
			err = system.CheckBashSyntax(ODR.Entrypoint)
			if err != nil {
				return err
			}
		}

		job, err := getAPIClient().Submit(ctx, spec, deal, nil)
		if err != nil {
			return err
		}

		cmd.Printf("%s\n", job.ID)
		if ODR.WaitForJobToFinish {
			resolver := getAPIClient().GetJobStateResolver()
			resolver.SetWaitTime(ODR.WaitForJobTimeoutSecs, time.Second*1)
			err = resolver.WaitUntilComplete(ctx, job.ID)
			if err != nil {
				return err
			}

			if ODR.WaitForJobToFinishAndPrintOutput {
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

		return nil
	},
}
