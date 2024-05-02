package docker

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
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
		Use:      "run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]",
		Short:    "Run a docker job on the network",
		Long:     runLong,
		Example:  runExample,
		Args:     cobra.MinimumNArgs(1),
		PreRunE:  hook.Chain(hook.RemoteCmdPreRunHooks, configflags.PreRun(dockerRunFlags)),
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return dockerRun(cmd, cmdArgs, opts)
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

func dockerRun(cmd *cobra.Command, cmdArgs []string, opts *DockerRunOptions) error {
	ctx := cmd.Context()

	image := cmdArgs[0]
	parameters := cmdArgs[1:]
	job, err := CreateJobDocker(image, parameters, opts)
	if err != nil {
		return fmt.Errorf("creating job: %w", err)
	}

	// Normalize and validate the job spec
	job.Normalize()
	if err := job.ValidateSubmission(); err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// TODO(forrest) [refactor]: this options is _almost_ useful. At present it marshals the entire
	// job spec to yaml, said spec cannot be used with `bacalhau job run` since it contains fields that
	// users are not permitted to set, like ID, Version, ModifyTime, State, etc.
	// The solution here is to have a "JobSubmission" type that is different from the actual job spec.
	if opts.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(job)
		if err != nil {
			return fmt.Errorf("converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	quiet := opts.RunTimeSettings.PrintJobIDOnly
	if !quiet {
		containsTag := dockerImageContainsTag(image)
		if !containsTag {
			cmd.PrintErrln("Using default tag: latest. Please specify a tag/digest for better reproducibility.")
		}
	}

	api := util.GetAPIClientV2(cmd)
	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	if len(resp.Warnings) > 0 {
		printWarnings(cmd, resp.Warnings)
	}

	if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, &opts.RunTimeSettings.RunTimeSettings, api); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

// TODO(forrest) [refactor]: dedupe from wasm_run
func printWarnings(cmd *cobra.Command, warnings []string) {
	cmd.Println("Warnings:")
	for _, warning := range warnings {
		cmd.Printf("\t* %s\n", warning)
	}
}

func CreateJobDocker(image string, parameters []string, opts *DockerRunOptions) (*models.Job, error) {
	engineSpec, err := models.DockerSpecBuilder(image).
		WithParameters(parameters...).
		WithWorkingDirectory(opts.WorkingDirectory).
		WithEntrypoint(opts.Entrypoint...).
		WithEnvironmentVariables(opts.SpecSettings.EnvVar...).Build()
	if err != nil {
		return nil, err
	}

	// TODO(forrest) [refactor]: this logic is duplicated in wasm_run
	resultPaths := make([]*models.ResultPath, 0, len(opts.SpecSettings.OutputVolumes))
	for name, path := range opts.SpecSettings.OutputVolumes {
		resultPaths = append(resultPaths, &models.ResultPath{
			Name: name,
			Path: path,
		})
	}

	task, err := models.NewTaskBuilder().
		Name("TODO").
		Engine(engineSpec).
		Publisher(opts.SpecSettings.Publisher.Value()).
		ResourcesConfig(&models.ResourcesConfig{
			CPU:    opts.ResourceSettings.CPU,
			Memory: opts.ResourceSettings.Memory,
			Disk:   opts.ResourceSettings.Disk,
			GPU:    opts.ResourceSettings.GPU,
		}).
		InputSources(opts.SpecSettings.Inputs.Values()...).
		ResultPaths(resultPaths...).
		Network(&models.NetworkConfig{
			Type:    opts.NetworkingSettings.Network,
			Domains: opts.NetworkingSettings.Domains,
		}).
		Timeouts(&models.TimeoutConfig{ExecutionTimeout: opts.SpecSettings.Timeout}).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	labels, err := parse.StringSliceToMap(opts.SpecSettings.Labels)
	if err != nil {
		return nil, fmt.Errorf("parseing job labels: %w", err)
	}

	constraints, err := parse.NodeSelector(opts.SpecSettings.Selector)
	if err != nil {
		return nil, fmt.Errorf("parseing job contstrints: %w", err)
	}
	job := &models.Job{
		Name:        "TODO",
		Namespace:   "TODO",
		Type:        models.JobTypeBatch,
		Priority:    0,
		Count:       opts.DealSettings.Concurrency,
		Constraints: constraints,
		Labels:      labels,
		Tasks:       []*models.Task{task},
	}

	return job, nil
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
