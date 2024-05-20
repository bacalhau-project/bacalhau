package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	clientv1 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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
	Entrypoint       []string
	WorkingDirectory string // Working directory for docker

	SpecSettings       *cliflags.SpecFlagSettings            // Setting for top level job spec fields.
	ResourceSettings   *cliflags.ResourceUsageSettings       // Settings for the jobs resource requirements.
	NetworkingSettings *cliflags.NetworkingFlagSettings      // Settings for the jobs networking.
	DealSettings       *cliflags.DealFlagSettings            // Settings for the jobs deal.
	RunTimeSettings    *cliflags.RunTimeSettingsWithDownload // Settings for running the job.
	DownloadSettings   *cliflags.DownloaderSettings          // Settings for running Download.

}

const (
	DefaultDockerRunWaitSeconds = 600
)

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Entrypoint:       nil,
		WorkingDirectory: "",

		SpecSettings:       cliflags.NewSpecFlagDefaultSettings(),
		ResourceSettings:   cliflags.NewDefaultResourceUsageSettings(),
		NetworkingSettings: cliflags.NewDefaultNetworkingFlagSettings(),
		DealSettings:       cliflags.NewDefaultDealFlagSettings(),
		DownloadSettings:   cliflags.NewDefaultDownloaderSettings(),
		RunTimeSettings:    cliflags.DefaultRunTimeSettingsWithDownload(),
	}
}

func NewCmd() *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "Run a docker job on the network (see run subcommand)",
	}

	dockerCmd.AddCommand(newDockerRunCmd())
	return dockerCmd
}

func newDockerRunCmd() *cobra.Command { //nolint:funlen
	opts := NewDockerRunOptions()

	dockerRunFlags := map[string][]configflags.Definition{
		"ipfs": configflags.IPFSFlags,
	}

	dockerRunCmd := &cobra.Command{
		Use:     "run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]",
		Short:   "Run a docker job on the network",
		Long:    runLong,
		Example: runExample,
		Args:    cobra.MinimumNArgs(1),
		// bind flags for this command to the config.
		PreRunE:  hook.Chain(hook.RemoteCmdPreRunHooks, configflags.PreRun(viper.GetViper(), dockerRunFlags)),
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig()
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create a v1 api client
			apiV1, err := util.GetAPIClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create v1 api client: %w", err)
			}
			// create a v2 api client
			apiV2, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create v2 api client: %w", err)
			}
			return dockerRun(cmd, cmdArgs, apiV1, apiV2, cfg, opts)
		},
	}

	dockerRunCmd.PersistentFlags().StringVarP(
		&opts.WorkingDirectory, "workdir", "w", opts.WorkingDirectory,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`,
	)

	dockerRunCmd.PersistentFlags().StringSliceVar(
		&opts.Entrypoint, "entrypoint", opts.Entrypoint,
		`Override the default ENTRYPOINT of the image`,
	)

	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.SpecFlags(opts.SpecSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.DealFlags(opts.DealSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.NewDownloadFlags(opts.DownloadSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.NetworkingFlags(opts.NetworkingSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.ResourceUsageFlags(opts.ResourceSettings))
	dockerRunCmd.PersistentFlags().AddFlagSet(cliflags.NewRunTimeSettingsFlagsWithDownload(opts.RunTimeSettings))

	if err := configflags.RegisterFlags(dockerRunCmd, dockerRunFlags); err != nil {
		util.Fatal(dockerRunCmd, err, 1)
	}

	return dockerRunCmd
}

func dockerRun(
	cmd *cobra.Command,
	cmdArgs []string,
	apiV1 *clientv1.APIClient,
	apiV2 clientv2.API,
	cfg types.BacalhauConfig,
	opts *DockerRunOptions,
) error {
	ctx := cmd.Context()

	image := cmdArgs[0]
	parameters := cmdArgs[1:]
	j, err := CreateJob(ctx, image, parameters, opts)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	if err := legacy_job.VerifyJob(ctx, j); err != nil {
		if _, ok := err.(*bacerrors.ImageNotFound); ok {
			return fmt.Errorf("docker image '%s' not found in the registry, or needs authorization", image)
		} else {
			return fmt.Errorf("verifying job: %s", err)
		}
	}

	quiet := opts.RunTimeSettings.PrintJobIDOnly
	if !quiet {
		containsTag := dockerImageContainsTag(image)
		if !containsTag {
			cmd.PrintErrln("Using default tag: latest. Please specify a tag/digest for better reproducibility.")
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

	if err := legacy_job.VerifyJob(ctx, j); err != nil {
		return fmt.Errorf("verifying job for submission: %w", err)
	}

	executingJob, err := apiV1.Submit(ctx, j)
	if err != nil {
		return fmt.Errorf("submitting job for execution: %w", err)
	}

	return printer.PrintJobExecutionLegacy(ctx, executingJob, cmd, opts.DownloadSettings, opts.RunTimeSettings, apiV1, apiV2, cfg.Node.IPFS)
}

// CreateJob creates a job object from the given command line arguments and options.
func CreateJob(ctx context.Context, image string, parameters []string, opts *DockerRunOptions) (*model.Job, error) {
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

	spec, err := legacy_job.MakeDockerSpec(
		image, opts.WorkingDirectory, opts.Entrypoint, opts.SpecSettings.EnvVar, parameters,
		legacy_job.WithResources(
			opts.ResourceSettings.CPU,
			opts.ResourceSettings.Memory,
			opts.ResourceSettings.Disk,
			opts.ResourceSettings.GPU,
		),
		legacy_job.WithNetwork(
			opts.NetworkingSettings.Network,
			opts.NetworkingSettings.Domains,
		),
		legacy_job.WithTimeout(opts.SpecSettings.Timeout),
		legacy_job.WithInputs(opts.SpecSettings.Inputs.Values()...),
		legacy_job.WithOutputs(outputs...),
		legacy_job.WithAnnotations(labels...),
		legacy_job.WithNodeSelector(nodeSelectorRequirements),
		legacy_job.WithDeal(
			opts.DealSettings.TargetingMode,
			opts.DealSettings.Concurrency,
		),
	)

	// Publisher is optional and we won't provide it if not specified
	p := opts.SpecSettings.Publisher.Value()
	if p != nil {
		spec.Publisher = p.Type //nolint:staticcheck
		spec.PublisherSpec = *p
	}

	if err != nil {
		return nil, err
	}

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
