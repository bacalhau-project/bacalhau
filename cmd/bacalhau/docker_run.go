package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/opts"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	dockerRunLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	dockerRunExample = templates.Examples(i18n.T(`
		# Run a Docker job, using the image 'dpokidov/imagemagick', with a CID mounted at /input_images and an output volume mounted at /outputs in the container. All flags after the '--' are passed directly into the container for execution.
		bacalhau docker run \
			-i src=ipfs://QmeZRGhe4PmjctYVSVHuEiA9oSXnqmYa4kQubSHgWbjv72,dst=/input_images \
			dpokidov/imagemagick:7.1.0-47-ubuntu \
			-- magick mogrify -resize 100x100 -quality 100 -path /outputs '/input_images/*.jpg'

		# Dry Run: check the job specification before submitting it to the bacalhau network
		bacalhau docker run --dry-run ubuntu echo hello

		# Save the job specification to a YAML file
		bacalhau docker run --dry-run ubuntu echo hello > job.yaml

		# Specify an image tag (default is 'latest' - using a specific tag other than 'latest' is recommended for reproducibility)
		bacalhau docker run ubuntu:bionic echo hello

		# Specify an image digest
		bacalhau docker run ubuntu@sha256:35b4f89ec2ee42e7e12db3d107fe6a487137650a2af379bbd49165a1494246ea echo hello
		`))
)

// DockerRunOptions declares the arguments accepted by the `docker run` command
type DockerRunOptions struct {
	Engine           string            // Executor - executor.Executor
	Verifier         string            // Verifier - verifier.Verifier
	Publisher        opts.PublisherOpt // Publisher - publisher.Publisher
	Inputs           opts.StorageOpt   // Array of inputs
	OutputVolumes    []string          // Array of output volumes in 'name:mount point' form
	Env              []string          // Array of environment variables
	IDOnly           bool              // Only print the job ID
	Concurrency      int               // Number of concurrent jobs to run
	Confidence       int               // Minimum number of nodes that must agree on a verification result
	MinBids          int               // Minimum number of bids before they will be accepted (at random)
	Timeout          float64           // Job execution timeout in seconds
	CPU              string
	Memory           string
	GPU              string
	Networking       model.Network
	NetworkDomains   []string
	WorkingDirectory string   // Working directory for docker
	Labels           []string // Labels for the job on the Bacalhau network (for searching)
	NodeSelector     string   // Selector (label query) to filter nodes on which this job can be executed

	Image      string   // Image to execute
	Entrypoint []string // Entrypoint to the docker image

	SkipSyntaxChecking bool // Verify the syntax using shellcheck

	DryRun bool // Don't submit the jobspec, print it to STDOUT

	RunTimeSettings RunTimeSettings // Settings for running the job

	DownloadFlags model.DownloaderSettings // Settings for running Download

	FilPlus bool // add a "filplus" label to the job to grab the attention of fil+ moderators
}

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Engine:             "docker",
		Verifier:           "noop",
		Publisher:          opts.NewPublisherOptFromSpec(model.PublisherSpec{Type: model.PublisherEstuary}),
		Inputs:             opts.StorageOpt{},
		OutputVolumes:      []string{},
		Env:                []string{},
		Concurrency:        1,
		Confidence:         0,
		MinBids:            0, // 0 means no minimum before bidding
		Timeout:            DefaultTimeout.Seconds(),
		CPU:                "",
		Memory:             "",
		GPU:                "",
		Networking:         model.NetworkNone,
		NetworkDomains:     []string{},
		SkipSyntaxChecking: false,
		WorkingDirectory:   "",
		Labels:             []string{},
		NodeSelector:       "",
		DownloadFlags:      *util.NewDownloadSettings(),
		RunTimeSettings:    *NewRunTimeSettings(),

		FilPlus: false,
	}
}

func newDockerCmd() *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:               "docker",
		Short:             "Run a docker job on the network (see run subcommand)",
		PersistentPreRunE: checkVersion,
	}

	dockerCmd.AddCommand(newDockerRunCmd())
	return dockerCmd
}

func newDockerRunCmd() *cobra.Command { //nolint:funlen
	ODR := NewDockerRunOptions()

	dockerRunCmd := &cobra.Command{
		Use:     "run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]",
		Short:   "Run a docker job on the network",
		Long:    dockerRunLong,
		Example: dockerRunExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return dockerRun(cmd, cmdArgs, ODR)
		},
	}

	// TODO: don't make jobEngine specifiable in the docker subcommand
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Engine, "engine", ODR.Engine,
		`What executor engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().StringVar(
		&ODR.Verifier, "verifier", ODR.Verifier,
		`What verification engine to use to run the job`,
	)
	dockerRunCmd.PersistentFlags().VarP(&ODR.Publisher, "publisher", "p",
		`Where to publish the result of the job`,
	)
	dockerRunCmd.PersistentFlags().VarP(&ODR.Inputs, "input", "i", inputUsageMsg)

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
	dockerRunCmd.PersistentFlags().Float64Var(
		&ODR.Timeout, "timeout", ODR.Timeout,
		`Job execution timeout in seconds (e.g. 300 for 5 minutes and 0.1 for 100ms)`,
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
	dockerRunCmd.PersistentFlags().Var(
		NetworkFlag(&ODR.Networking), "network",
		`Networking capability required by the job`,
	)
	dockerRunCmd.PersistentFlags().StringArrayVar(
		&ODR.NetworkDomains, "domain", ODR.NetworkDomains,
		`Domain(s) that the job needs to access (for HTTP networking)`,
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

	dockerRunCmd.PersistentFlags().StringVarP(
		&ODR.NodeSelector, "selector", "s", ODR.NodeSelector,
		`Selector (label query) to filter nodes on which this job can be executed, supports '=', '==', and '!='.(e.g. -s key1=value1,key2=value2). Matching objects must satisfy all of the specified label constraints.`, //nolint:lll // Documentation, ok if long.
	)

	dockerRunCmd.PersistentFlags().BoolVar(
		&ODR.FilPlus, "filplus", ODR.FilPlus,
		`Mark the job as a candidate for moderation for FIL+ rewards.`,
	)

	dockerRunCmd.PersistentFlags().AddFlagSet(NewRunTimeSettingsFlags(&ODR.RunTimeSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(NewIPFSDownloadFlags(&ODR.DownloadFlags))

	return dockerRunCmd
}

func dockerRun(cmd *cobra.Command, cmdArgs []string, ODR *DockerRunOptions) error {
	ctx := cmd.Context()

	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

	j, err := CreateJob(ctx, cmdArgs, ODR)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error creating job: %s", err), 1)
		return nil
	}
	err = jobutils.VerifyJob(ctx, j)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error verifying job: %s", err), 1)
		return nil
	}

	quiet := ODR.RunTimeSettings.PrintJobIDOnly
	if !quiet {
		dockerEngine, err := model.DecodeJobSpecDocker(j.Spec.EngineSpec.Spec)
		if err != nil {
			Fatal(cmd, err.Error(), 1)
		}
		containsTag := DockerImageContainsTag(dockerEngine.Image)
		if !containsTag {
			cmd.Printf("Using default tag: latest. Please specify a tag/digest for better reproducibility.\n")
		}
	}

	if ODR.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error converting job to yaml: %s", err), 1)
			return nil
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	return ExecuteJob(ctx,
		cm,
		cmd,
		j,
		ODR.RunTimeSettings,
		ODR.DownloadFlags,
	)
}

// CreateJob creates a job object from the given command line arguments and options.
func CreateJob(ctx context.Context, cmdArgs []string, odr *DockerRunOptions) (*model.Job, error) { //nolint:funlen,gocyclo
	odr.Image = cmdArgs[0]
	odr.Entrypoint = cmdArgs[1:]

	swarmAddresses := odr.DownloadFlags.IPFSSwarmAddrs

	if swarmAddresses == "" {
		swarmAddresses = strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ",")
	}

	odr.DownloadFlags = model.DownloaderSettings{
		Timeout:        odr.DownloadFlags.Timeout,
		OutputDir:      odr.DownloadFlags.OutputDir,
		IPFSSwarmAddrs: swarmAddresses,
	}

	engineType, err := model.ParseEngine(odr.Engine)
	if err != nil {
		return &model.Job{}, err
	}

	verifierType, err := model.ParseVerifier(odr.Verifier)
	if err != nil {
		return &model.Job{}, err
	}

	if len(odr.WorkingDirectory) > 0 {
		err = system.ValidateWorkingDir(odr.WorkingDirectory)

		if err != nil {
			return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal:")
		}
	}

	labels := odr.Labels

	if odr.FilPlus {
		labels = append(labels, "filplus")
	}

	j, err := jobutils.ConstructDockerJob(
		ctx,
		model.APIVersionLatest(),
		engineType,
		verifierType,
		odr.Publisher.Value(),
		odr.CPU,
		odr.Memory,
		odr.GPU,
		odr.Networking,
		odr.NetworkDomains,
		odr.Inputs.Values(),
		odr.OutputVolumes,
		odr.Env,
		odr.Entrypoint,
		odr.Image,
		odr.Concurrency,
		odr.Confidence,
		odr.MinBids,
		odr.Timeout,
		labels,
		odr.NodeSelector,
		odr.WorkingDirectory,
	)
	if err != nil {
		return &model.Job{}, errors.Wrap(err, "CreateJobSpecAndDeal")
	}

	return j, nil
}
