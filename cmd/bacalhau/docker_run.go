package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/filecoin-project/bacalhau/pkg/version"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"
)

const (
	CompleteStatus = "Complete"
)

var (
	dockerRunLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	dockerRunExample = templates.Examples(i18n.T(`
		# Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /outputs in the container.
		# All flags after the '--' are passed directly into the container for execution.
		bacalhau docker run \
			-v QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72:/input_images \
			dpokidov/imagemagick:7.1.0-47-ubuntu \
			-- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'
			
		# Dry Run: Check the job specification before submitting it to the bacalhau network
		bacalhau docker run --dry-run ubuntu echo hello

		saving the job specification to a yaml file
		bacalhau docker run --dry-run ubuntu echo hello > job.yaml
		`))

	// Set Defaults (probably a better way to do this)
	ODR = NewDockerRunOptions()
)

// DockerRunOptions declares the arguments accepted by the `docker run` command
type DockerRunOptions struct {
	Engine           string   // Executor - executor.Executor
	Verifier         string   // Verifier - verifier.Verifier
	Publisher        string   // Publisher - publisher.Publisher
	Inputs           []string // Array of input CIDs
	InputUrls        []string // Array of input URLs (will be copied to IPFS)
	InputVolumes     []string // Array of input volumes in 'CID:mount point' form
	OutputVolumes    []string // Array of output volumes in 'name:mount point' form
	Env              []string // Array of environment variables
	IDOnly           bool     // Only print the job ID
	Concurrency      int      // Number of concurrent jobs to run
	Confidence       int      // Minimum number of nodes that must agree on a verification result
	MinBids          int      // Minimum number of bids before they will be accepted (at random)
	CPU              string
	Memory           string
	GPU              string
	WorkingDirectory string   // Working directory for docker
	Labels           []string // Labels for the job on the Bacalhau network (for searching)

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking bool // Verify the syntax using shellcheck

	DryRun bool // Don't submit the jobspec, print it to STDOUT

	RunTimeSettings RunTimeSettings // Settings for running the job

	DownloadFlags ipfs.IPFSDownloadSettings // Settings for running Download

	ShardingGlobPattern string
	ShardingBasePath    string
	ShardingBatchSize   int
}

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Engine:             "docker",
		Verifier:           "noop",
		Publisher:          "estuary",
		Inputs:             []string{},
		InputUrls:          []string{},
		InputVolumes:       []string{},
		OutputVolumes:      []string{},
		Env:                []string{},
		Concurrency:        1,
		Confidence:         0,
		MinBids:            0, // 0 means no minimum before bidding
		CPU:                "",
		Memory:             "",
		GPU:                "",
		SkipSyntaxChecking: false,
		WorkingDirectory:   "",
		Labels:             []string{},
		DownloadFlags:      *ipfs.NewIPFSDownloadSettings(),
		RunTimeSettings:    *NewRunTimeSettings(),

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
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Publisher, "publisher", ODR.Publisher,
		`What publisher engine to use to publish the job results`,
	)
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Inputs, "inputs", "i", ODR.Inputs,
		`CIDs to use on the job. Mounts them at '/inputs' in the execution.`,
	)

	//nolint:lll // Documentation, ok if long.
	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.InputUrls, "input-urls", "u", ODR.InputUrls,
		`URL of the input data volumes downloaded from a URL source. Mounts data at '/inputs' (e.g. '-u http://foo.com/bar.tar.gz'
		mounts 'bar.tar.gz' at '/inputs/bar.tar.gz'). URL accept any valid URL supported by the 'wget' command,
		and supports both HTTP and HTTPS.`,
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
		&ODR.Confidence, "confidence", ODR.Confidence,
		`The minimum number of nodes that must agree on a verification result`,
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

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.DryRun, "dry-run", ODR.DryRun,
		`Do not submit the job, but instead print out what will be submitted`,
	)

	dockerRunCmd.PersistentFlags().StringVarP(
		&ODR.WorkingDirectory, "workdir", "w", ODR.WorkingDirectory,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVarP(
		&ODR.Labels, "labels", "l", ODR.Labels,
		`List of labels for the job. Enter multiple in the format '-l a -l 2'. All characters not matching /a-zA-Z0-9_:|-/ and all emojis will be stripped.`, //nolint:lll // Documentation, ok if long.
	)

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

	dockerRunCmd.PersistentFlags().AddFlagSet(NewRunTimeSettingsFlags(&ODR.RunTimeSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(NewIPFSDownloadFlags(&ODR.DownloadFlags))
}

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Run a docker job on the network (see run subcommand)",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Check that the server version is compatible with the client version
		serverVersion, _ := GetAPIClient().Version(cmd.Context()) // Ok if this fails, version validation will skip
		if err := ensureValidVersion(cmd.Context(), version.Get(), serverVersion); err != nil {
			cmd.Println(err.Error())
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
	PreRun:  applyPorcelainLogLevel,
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrect that cmd is unused.
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/dockerRun")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		j, err := CreateJob(ctx, cmdArgs, ODR)
		if err != nil {
			Fatal(fmt.Sprintf("Error creating job: %s", err), 1)
			return nil
		}
		err = jobutils.VerifyJob(ctx, j)
		if err != nil {
			if _, ok := err.(*bacerrors.ImageNotFound); ok {
				Fatal(fmt.Sprintf("Docker image '%s' not found in the registry, or needs authorization.", j.Spec.Docker.Image), 1)
				return nil
			} else {
				Fatal(fmt.Sprintf("Error verifying job: %s", err), 1)
				return nil
			}
		}
		if ODR.DryRun {
			// Converting job to yaml
			var yamlBytes []byte
			yamlBytes, err = yaml.Marshal(j)
			if err != nil {
				Fatal(fmt.Sprintf("Error converting job to yaml: %s", err), 1)
				return nil
			}
			cmd.Print(string(yamlBytes))
			return nil
		}

		err = ExecuteJob(ctx,
			cm,
			cmd,
			j,
			ODR.RunTimeSettings,
			ODR.DownloadFlags,
			nil,
		)
		if err != nil {
			Fatal(fmt.Sprintf("Error executing job: %s", err), 1)
			return nil
		}

		return nil
	},
}

func CreateJob(ctx context.Context,
	cmdArgs []string,
	odr *DockerRunOptions) (*model.Job, error) {
	//nolint:ineffassign,staticcheck
	_, span := system.GetTracer().Start(ctx, "cmd/bacalhau/dockerRun.ProcessAndExecuteJob")
	defer span.End()

	odr.Image = cmdArgs[0]
	odr.Entrypoint = cmdArgs[1:]

	odr.DownloadFlags = ipfs.IPFSDownloadSettings{
		TimeoutSecs:    odr.DownloadFlags.TimeoutSecs,
		OutputDir:      odr.DownloadFlags.OutputDir,
		IPFSSwarmAddrs: strings.Join(system.Envs[system.Production].IPFSSwarmAddresses, ","),
	}

	engineType, err := model.ParseEngine(odr.Engine)
	if err != nil {
		return &model.Job{}, err
	}

	verifierType, err := model.ParseVerifier(odr.Verifier)
	if err != nil {
		return &model.Job{}, err
	}

	publisherType, err := model.ParsePublisher(odr.Publisher)
	if err != nil {
		return &model.Job{}, err
	}

	for _, i := range odr.Inputs {
		odr.InputVolumes = append(odr.InputVolumes, fmt.Sprintf("%s:/inputs", i))
	}

	if len(odr.WorkingDirectory) > 0 {
		err = system.ValidateWorkingDir(odr.WorkingDirectory)

		if err != nil {
			return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal:")
		}
	}

	j, err := jobutils.ConstructDockerJob(
		model.APIVersionLatest(),
		engineType,
		verifierType,
		publisherType,
		odr.CPU,
		odr.Memory,
		odr.GPU,
		odr.InputUrls,
		odr.InputVolumes,
		odr.OutputVolumes,
		odr.Env,
		odr.Entrypoint,
		odr.Image,
		odr.Concurrency,
		odr.Confidence,
		odr.MinBids,
		odr.Labels,
		odr.WorkingDirectory,
		odr.ShardingGlobPattern,
		odr.ShardingBasePath,
		odr.ShardingBatchSize,
		doNotTrack,
	)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal")
	}

	return j, nil
}
