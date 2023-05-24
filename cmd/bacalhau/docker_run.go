package bacalhau

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/opts"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/util"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/specs/engine/docker"
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

func dockerRun(cmd *cobra.Command, cmdArgs []string, opts *DockerRunOptions) error {
	ctx := cmd.Context()

	// TODO not a fan of storing things in the context like this, half the receives don't use it, and accept a context
	// anyways - which they could pull this from. If we want to keep this perhaps it could be a global somewhere.
	cm := ctx.Value(systemManagerKey).(*system.CleanupManager)

	// also handles validation
	dockerJob, err := createDockerJob(ctx, cmdArgs, opts)
	if err != nil {
		return fmt.Errorf("creating docker job: %w", err)
	}

	if !opts.RunTimeSettings.PrintJobIDOnly && !DockerImageContainsTag(dockerJob.DockerSpec.Image) {
		return fmt.Errorf("image %s does not contain tag, please specify a tag/digest", dockerJob.DockerSpec.Image)
	}

	if opts.DryRun {
		// Converting docker job to Spec then to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(dockerJob)
		if err != nil {
			return fmt.Errorf("converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}
	return ExecuteDockerJob(ctx, cm, cmd, dockerJob, &ExecutionSettings{
		Runtime:  opts.RunTimeSettings,
		Download: opts.DownloadFlags,
	})
}

func createDockerJob(ctx context.Context, cmdArgs []string, opts *DockerRunOptions) (*model.DockerJob, error) {
	verifierSpec, err := model.ParseVerifier(opts.Verifier)
	if err != nil {
		return nil, err
	}

	outputSpec, err := jobutils.BuildJobOutputs(ctx, opts.OutputVolumes)
	if err != nil {
		return nil, err
	}

	nodeSelectors, err := jobutils.ParseNodeSelector(opts.NodeSelector)
	if err != nil {
		return nil, err
	}

	labels := opts.Labels
	if opts.FilPlus {
		labels = append(labels, "filplus")
	}

	swarmAddresses := opts.DownloadFlags.IPFSSwarmAddrs
	if swarmAddresses == "" {
		swarmAddresses = strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ",")
	}
	opts.DownloadFlags = model.DownloaderSettings{
		Timeout:        opts.DownloadFlags.Timeout,
		OutputDir:      opts.DownloadFlags.OutputDir,
		IPFSSwarmAddrs: swarmAddresses,
	}

	out := &model.DockerJob{
		// TODO this could be different than the api version as it only relates to docker jobs.
		APIVersion: model.APIVersionLatest(),
		DockerSpec: docker.DockerEngineSpec{
			Image:                cmdArgs[0],
			Entrypoint:           cmdArgs[1:],
			EnvironmentVariables: opts.Env,
			WorkingDirectory:     opts.WorkingDirectory,
		},
		PublisherSpec: opts.Publisher.Value(),
		VerifierSpec:  verifierSpec,
		ResourceConfig: model.ResourceUsageConfig{
			CPU:    opts.CPU,
			Memory: opts.Memory,
			GPU:    opts.GPU,
			// TODO this is unspecified on CLI
			// Disk:   opts.Disk?,
		},
		NetworkConfig: model.NetworkConfig{
			Type:    opts.Networking,
			Domains: opts.NetworkDomains,
		},
		Inputs:  opts.Inputs.Values(),
		Outputs: outputSpec,
		DealSpec: model.Deal{
			Concurrency: opts.Concurrency,
			Confidence:  opts.Confidence,
			MinBids:     opts.MinBids,
		},
		NodeSelectors: nodeSelectors,
		Timeout:       opts.Timeout,
		Annotations:   opts.Labels,
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}
