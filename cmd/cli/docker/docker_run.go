package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags"
	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
	"github.com/bacalhau-project/bacalhau/cmd/util/opts"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	runLong = templates.LongDesc(i18n.T(`
		Runs a job using the Docker executor on the node.
		`))

	//nolint:lll // Documentation
	runExample = templates.Examples(i18n.T(`
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
	WorkingDirectory string // Working directory for docker

	SpecSettings       flags.SpecFlagSettings       // Setting for top level job spec fields.
	ResourceSettings   flags.ResourceUsageSettings  // Settings for the jobs resource requirements.
	NetworkingSettings flags.NetworkingFlagSettings // Settings for the jobs networking.
	DealSettings       flags.DealFlagSettings       // Settings for the jobs deal.
	RunTimeSettings    flags.RunTimeSettings        // Settings for running the job.
	DownloadSettings   flags.DownloaderSettings     // Settings for running Download.

}

const (
	DefaultDockerRunWaitSeconds = 600
)

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		SpecSettings: flags.SpecFlagSettings{
			Verifier: model.VerifierNoop.String(),
			// TODO most users would probably prefer IPFS
			Publisher: opts.NewPublisherOptFromSpec(model.PublisherSpec{Type: model.PublisherEstuary}),
			Inputs:    opts.StorageOpt{},
			// TODO validate this works or make it a concrete spec type
			OutputVolumes: []string{"outputs:/outputs"},
			EnvVar:        []string{},
			Timeout:       jobutils.DefaultTimeout.Seconds(),
			Labels:        []string{},
			Selector:      "",
			DoNotTrack:    false,
		},
		// Below is common across wasm
		ResourceSettings: flags.ResourceUsageSettings{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		NetworkingSettings: flags.NetworkingFlagSettings{
			Network: model.NetworkNone,
			Domains: []string{},
		},
		DealSettings: flags.DealFlagSettings{
			Concurrency:   1,
			Confidence:    0,
			MinBids:       0, // 0 means no minimum before bidding
			TargetingMode: model.TargetAny,
		},
		WorkingDirectory: "",

		DownloadSettings: flags.DownloaderSettings{
			Timeout:        model.DefaultIPFSTimeout,
			IPFSSwarmAddrs: strings.Join(system.Envs[system.GetEnvironment()].IPFSSwarmAddresses, ","),
			OutputDir:      "",
			SingleFile:     "",
			LocalIPFS:      false,
			Raw:            false,
		},
		RunTimeSettings: flags.RunTimeSettings{
			AutoDownloadResults:   false,
			WaitForJobToFinish:    true,
			WaitForJobTimeoutSecs: DefaultDockerRunWaitSeconds,
			IPFSGetTimeOut:        10,
			IsLocal:               false,
			PrintJobIDOnly:        false,
			PrintNodeDetails:      false,
			Follow:                false,
			SkipSyntaxChecking:    false,
			DryRun:                false,
		},
	}
}

func NewCmd() *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:               "docker",
		Short:             "Run a docker job on the network (see run subcommand)",
		PersistentPreRunE: handler.CheckVersion,
	}

	dockerCmd.AddCommand(newDockerRunCmd())
	return dockerCmd
}

func newDockerRunCmd() *cobra.Command { //nolint:funlen
	opts := NewDockerRunOptions()

	dockerRunCmd := &cobra.Command{
		Use:     "run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]",
		Short:   "Run a docker job on the network",
		Long:    runLong,
		Example: runExample,
		Args:    cobra.MinimumNArgs(1),
		PreRun:  handler.ApplyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			if err := dockerRun(cmd, cmdArgs, opts); err != nil {
				handler.Fatal(cmd, err, 1)
			}
			return nil
		},
	}

	dockerRunCmd.PersistentFlags().StringVarP(
		&opts.WorkingDirectory, "workdir", "w", opts.WorkingDirectory,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().AddFlagSet(flags.SpecFlags(&opts.SpecSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(flags.DealFlags(&opts.DealSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(flags.NewDownloadFlags(&opts.DownloadSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(flags.NetworkingFlags(&opts.NetworkingSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(flags.ResourceUsageFlags(&opts.ResourceSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(flags.NewRunTimeSettingsFlags(&opts.RunTimeSettings))

	return dockerRunCmd
}

func dockerRun(cmd *cobra.Command, cmdArgs []string, opts *DockerRunOptions) error {
	ctx := cmd.Context()

	j, err := CreateJob(ctx, cmdArgs, opts)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	if err := jobutils.VerifyJob(ctx, j); err != nil {
		if _, ok := err.(*bacerrors.ImageNotFound); ok {
			return fmt.Errorf("docker image '%s' not found in the registry, or needs authorization", j.Spec.Docker.Image)
		} else {
			return fmt.Errorf("verifying job: %s", err)
		}
	}

	quiet := opts.RunTimeSettings.PrintJobIDOnly
	if !quiet {
		containsTag := dockerImageContainsTag(j.Spec.Docker.Image)
		if !containsTag {
			cmd.Printf("Using default tag: latest. Please specify a tag/digest for better reproducibility.\n")
		}
	}

	if opts.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			return fmt.Errorf("converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	executingJob, err := handler.ExecuteJob(ctx, j, opts.RunTimeSettings)
	if err != nil {
		return err
	}

	return printer.PrintJobExecution(ctx, executingJob, cmd, opts.DownloadSettings, opts.RunTimeSettings, handler.GetAPIClient(ctx))
}

// CreateJob creates a job object from the given command line arguments and options.
func CreateJob(ctx context.Context, cmdArgs []string, opts *DockerRunOptions) (*model.Job, error) { //nolint:funlen,gocyclo
	image := cmdArgs[0]
	entrypoint := cmdArgs[1:]

	verifierType, err := model.ParseVerifier(opts.SpecSettings.Verifier)
	if err != nil {
		return nil, err
	}

	outputs, err := parse.JobOutputs(ctx, opts.SpecSettings.OutputVolumes)
	if err != nil {
		return nil, err
	}

	nodeSelectorRequirements, err := parse.NodeSelector(opts.SpecSettings.Selector)
	if err != nil {
		return nil, err
	}

	labels, err := parse.Labels(ctx, opts.SpecSettings.Labels)
	if err != nil {
		return nil, err
	}

	spec, err := jobutils.MakeDockerSpec(
		image, entrypoint, opts.SpecSettings.EnvVar, opts.WorkingDirectory,
		jobutils.WithVerifier(verifierType),
		jobutils.WithPublisher(opts.SpecSettings.Publisher.Value()),
		jobutils.WithResources(
			opts.ResourceSettings.CPU,
			opts.ResourceSettings.Memory,
			opts.ResourceSettings.Disk,
			opts.ResourceSettings.GPU,
		),
		jobutils.WithNetwork(
			opts.NetworkingSettings.Network,
			opts.NetworkingSettings.Domains,
		),
		jobutils.WithTimeout(opts.SpecSettings.Timeout),
		jobutils.WithInputs(opts.SpecSettings.Inputs.Values()...),
		jobutils.WithOutputs(outputs...),
		jobutils.WithAnnotations(labels...),
		jobutils.WithNodeSelector(nodeSelectorRequirements),
		jobutils.WithDeal(
			opts.DealSettings.TargetingMode,
			opts.DealSettings.Concurrency,
			opts.DealSettings.Confidence,
			opts.DealSettings.MinBids,
		),
	)

	return &model.Job{
		APIVersion: model.APIVersionLatest().String(),
		Spec:       spec,
	}, nil
}

// dockerImageContainsTag checks if the image contains a tag or a digest
func dockerImageContainsTag(image string) bool {
	if strings.Contains(image, ":") {
		return true
	}
	if strings.Contains(image, "@") {
		return true
	}
	return false
}
