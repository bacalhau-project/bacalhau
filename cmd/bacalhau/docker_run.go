package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/version"

	"github.com/filecoin-project/bacalhau/pkg/publisher"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

const (
	CompleteStatus              = "Complete"
	DefaultDockerRunWaitSeconds = 600
)

var (
	dockerRunLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	dockerRunExample = templates.Examples(i18n.T(`
		# Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /output_images in the container.
		# All flags after the '--' are passed directly into the container for execution.
		bacalhau docker run \
		-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
		-o results:/output_images \
		dpokidov/imagemagick \
		-- magick mogrify -resize 100x100 -quality 100 -path /output_images /input_images/*.jpg`))

	// Set Defaults (probably a better way to do this)
	ODR = NewDockerRunOptions()
)

// DockerRunOptions declares the arguments accepted by the `docker run` command
type DockerRunOptions struct {
	Engine        string   // Executor - executor.Executor
	Verifier      string   // Verifier - verifier.Verifier
	Publisher     string   // Publisher - publisher.Publisher
	Inputs        []string // Array of input CIDs
	InputUrls     []string // Array of input URLs (will be copied to IPFS)
	InputVolumes  []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes []string // Array of output volumes in 'name:mount point' form
	Env           []string // Array of environment variables
	Concurrency   int      // Number of concurrent jobs to run
	MinBids       int      // Minimum number of bids before they will be accepted (at random)
	CPU           string
	Memory        string
	GPU           string
	WorkingDir    string   // Working directory for docker
	Labels        []string // Labels for the job on the Bacalhau network (for searching)

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking               bool                      // Verify the syntax using shellcheck
	WaitForJobToFinish               bool                      // Wait for the job to execute before exiting
	WaitForJobToFinishAndPrintOutput bool                      // Wait for the job to execute, and print the results before exiting
	WaitForJobTimeoutSecs            int                       // Job time out in seconds
	IPFSGetTimeOut                   int                       // Timeout for IPFS in seconds
	IsLocal                          bool                      // Job should be executed locally
	DockerRunDownloadFlags           ipfs.IPFSDownloadSettings // Settings for running Download

	ShardingGlobPattern string
	ShardingBasePath    string
	ShardingBatchSize   int
}

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Engine:                           "docker",
		Verifier:                         "noop",
		Publisher:                        "ipfs",
		Inputs:                           []string{},
		InputUrls:                        []string{},
		InputVolumes:                     []string{},
		OutputVolumes:                    []string{},
		Env:                              []string{},
		Concurrency:                      1,
		MinBids:                          0, // 0 means no minimum before bidding
		CPU:                              "",
		Memory:                           "",
		GPU:                              "",
		SkipSyntaxChecking:               false,
		WorkingDir:                       "",
		Labels:                           []string{},
		WaitForJobToFinish:               false,
		WaitForJobToFinishAndPrintOutput: false,
		WaitForJobTimeoutSecs:            DefaultDockerRunWaitSeconds,
		DockerRunDownloadFlags: ipfs.IPFSDownloadSettings{
			TimeoutSecs:    10,
			OutputDir:      ".",
			IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
		},
		IPFSGetTimeOut: 10,
		IsLocal:        false,

		ShardingGlobPattern: "",
		ShardingBasePath:    "/inputs",
		ShardingBatchSize:   1,
	}
}

func init() { //nolint:gochecknoinits,funlen // Using init in cobra command is idomatic
	dockerCmd.AddCommand(dockerRunCmd)

	// TODO: don't make jobEngine specifiable in the docker subcommand
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Engine, "engine", ODR.Engine,
		`What executor engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Verifier, "verifier", ODR.Verifier,
		`What verification engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Inputs, "inputs", "i", ODR.Inputs,
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	//nolint:lll // Documentation, ok if long.
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.InputUrls, "input-urls", "u", ODR.InputUrls,
		`URL:path of the input data volumes downloaded from a URL source. Mounts data at 'path' (e.g. '-u http://foo.com/bar.tar.gz:/app/bar.tar.gz'
		mounts 'http://foo.com/bar.tar.gz' at '/app/bar.tar.gz'). URL can specify a port number (e.g. 'https://foo.com:443/bar.tar.gz:/app/bar.tar.gz')
		and supports HTTP and HTTPS.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.InputVolumes, "input-volumes", "v", ODR.InputVolumes,
		`CID:path of the input data volumes, if you need to set the path of the mounted data.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.OutputVolumes, "output-volumes", "o", ODR.OutputVolumes,
		`name:path of the output data volumes. 'outputs:/outputs' is always added.`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Env, "env", "e", ODR.Env,
		`The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)`,
	)
	dockerRunCmd.PersistentFlags().IntVarP(
		&ODR.Concurrency, "concurrency", "c", ODR.Concurrency,
		`How many nodes should run the job`,
	)
	dockerRunCmd.PersistentFlags().IntVar(
		&ODR.MinBids, "min-bids", ODR.MinBids,
		`Minimum number of bids that must be received before concurrency-many bids will be accepted (at random)`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.CPU, "cpu", ODR.CPU,
		`Job CPU cores (e.g. 500m, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Memory, "memory", ODR.Memory,
		`Job Memory requirement (e.g. 500Mb, 2Gb, 8Gb).`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.GPU, "gpu", ODR.GPU,
		`Job GPU requirement (e.g. 1, 2, 8).`,
	)
	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.SkipSyntaxChecking, "skip-syntax-checking", ODR.SkipSyntaxChecking,
		`Skip having 'shellchecker' verify syntax of the command`,
	)

	dockerRunCmd.PersistentFlags().StringVarP(
		&ODR.WorkingDir, "workdir", "w", ODR.WorkingDir,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Labels, "labels", "l", ODR.Labels,
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.WaitForJobToFinish, "wait", ODR.WaitForJobToFinish,
		`Wait for the job to finish.`,
	)

	dockerRunCmd.PersistentFlags().IntVarP(
		&ODR.IPFSGetTimeOut, "gettimeout", "g", ODR.IPFSGetTimeOut,
		`Timeout for getting the results of a job in --wait`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.IsLocal, "local", ODR.IsLocal,
		`Run the job locally. Docker is required`,
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.WaitForJobToFinishAndPrintOutput, "download", ODR.WaitForJobToFinishAndPrintOutput,
		`Download the results and print stdout once the job has completed (implies --wait).`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&ODR.WaitForJobTimeoutSecs, "wait-timeout-secs", ODR.WaitForJobTimeoutSecs,
		`When using --wait, how many seconds to wait for the job to complete before giving up.`,
	)

	dockerRunCmd.Flags().IntVar(&ODR.DockerRunDownloadFlags.TimeoutSecs, "download-timeout-secs",
		ODR.DockerRunDownloadFlags.TimeoutSecs, "Timeout duration for IPFS downloads.")
	dockerRunCmd.Flags().StringVar(&ODR.DockerRunDownloadFlags.OutputDir, "output-dir",
		ODR.DockerRunDownloadFlags.OutputDir, "Directory to write the output to.")
	dockerRunCmd.Flags().StringVar(&ODR.DockerRunDownloadFlags.IPFSSwarmAddrs, "ipfs-swarm-addrs",
		ODR.DockerRunDownloadFlags.IPFSSwarmAddrs, "Comma-separated list of IPFS nodes to connect to.")

	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.ShardingGlobPattern, "sharding-glob-pattern", ODR.ShardingGlobPattern,
		`Use this pattern to match files to be sharded.`,
	)

	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.ShardingBasePath, "sharding-base-path", ODR.ShardingBasePath,
		`Where the sharding glob pattern starts from - useful when you have multiple volumes.`,
	)

	dockerRunCmd.PersistentFlags().IntVar(
		&ODR.ShardingBatchSize, "sharding-batch-size", ODR.ShardingBatchSize,
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
	PostRun: func(cmd *cobra.Command, args []string) {},
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := context.Background()

		ODR.Image = cmdArgs[0]
		ODR.Entrypoint = cmdArgs[1:]

		ODR.DockerRunDownloadFlags = ipfs.IPFSDownloadSettings{
			TimeoutSecs:    ODR.DockerRunDownloadFlags.TimeoutSecs,
			OutputDir:      ODR.DockerRunDownloadFlags.OutputDir,
			IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
		}

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

		publisherType, err := publisher.ParsePublisherType(ODR.Publisher)
		if err != nil {
			return err
		}

		for _, i := range ODR.Inputs {
			ODR.InputVolumes = append(ODR.InputVolumes, fmt.Sprintf("%s:/inputs", i))
		}

		if len(ODR.WorkingDir) > 0 {
			err = system.ValidateWorkingDir(ODR.WorkingDir)

			if err != nil {
				return fmt.Errorf("invalid working directory: %s", err)
			}
		}

		jobSpec, jobDeal, err := jobutils.ConstructDockerJob(
			engineType,
			verifierType,
			publisherType,
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
			ODR.MinBids,
			ODR.Labels,
			ODR.WorkingDir,
			ODR.ShardingGlobPattern,
			ODR.ShardingBasePath,
			ODR.ShardingBatchSize,
			doNotTrack,
		)
		if err != nil {
			return fmt.Errorf("error executing job: %s", err)
		}

		err = ExecuteJob(ctx,
			cm,
			cmd,
			jobSpec,
			jobDeal,
			ODR.IsLocal,
			ODR.WaitForJobToFinish,
			ODR.DockerRunDownloadFlags)

		if err != nil {
			return fmt.Errorf("error executing job: %s", err)
		}

		return nil
	},
}
