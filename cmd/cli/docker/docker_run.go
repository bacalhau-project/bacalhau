package docker

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/cli/helpers"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
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
	Entrypoint           []string
	WorkingDirectory     string
	EnvironmentVariables []string

	JobSettings      *cliflags.JobSettings
	TaskSettings     *cliflags.TaskSettings
	RunTimeSettings  *cliflags.RunTimeSettings
	DownloadSettings *cliflags.DownloaderSettings
}

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Entrypoint:       nil,
		WorkingDirectory: "",

		JobSettings:      cliflags.DefaultJobSettings(),
		TaskSettings:     cliflags.DefaultTaskSettings(),
		DownloadSettings: cliflags.DefaultDownloaderSettings(),
		RunTimeSettings:  cliflags.DefaultRunTimeSettings(),
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

	// flags with a corresponding config value via env vars, config file
	configuredFlags := map[string][]configflags.Definition{
		"ipfs": configflags.IPFSFlags,
	}

	cmd := &cobra.Command{
		Use:      "run [flags] IMAGE[:TAG|@DIGEST] [COMMAND] [ARG...]",
		Short:    "Run a docker job on the network",
		Long:     runLong,
		Example:  runExample,
		Args:     cobra.MinimumNArgs(1),
		PreRunE:  hook.Chain(hook.RemoteCmdPreRunHooks, configflags.PreRun(configuredFlags)),
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return run(cmd, cmdArgs, opts)
		},
	}

	// register config-based flags.
	if err := configflags.RegisterFlags(cmd, configuredFlags); err != nil {
		util.Fatal(cmd, err, 1)
	}

	// register common flags.
	cliflags.RegisterJobFlags(cmd, opts.JobSettings)
	cliflags.RegisterTaskFlags(cmd, opts.TaskSettings)
	cliflags.RegisterDownloadFlags(cmd, opts.DownloadSettings)
	cliflags.RegisterRunTimeFlags(cmd, opts.RunTimeSettings)

	// register flags unique to docker.
	dockerFlags := pflag.NewFlagSet("docker", pflag.ContinueOnError)
	dockerFlags.StringVarP(&opts.WorkingDirectory, "workdir", "w", opts.WorkingDirectory,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`)
	dockerFlags.StringSliceVar(&opts.Entrypoint, "entrypoint", opts.Entrypoint,
		`Override the default ENTRYPOINT of the image`)
	dockerFlags.StringSliceVarP(&opts.EnvironmentVariables, "env", "e", opts.EnvironmentVariables,
		"The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)")

	cmd.PersistentFlags().AddFlagSet(dockerFlags)

	return cmd
}

func run(cmd *cobra.Command, args []string, opts *DockerRunOptions) error {
	ctx := cmd.Context()

	job, err := build(args, opts)
	if err != nil {
		return err
	}

	if opts.RunTimeSettings.DryRun {
		out, err := helpers.JobToYaml(job)
		if err != nil {
			return err
		}
		cmd.Print(out)
		return nil
	}

	api := util.GetAPIClientV2(cmd)
	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	if len(resp.Warnings) > 0 {
		helpers.PrintWarnings(cmd, resp.Warnings)
	}

	if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, opts.RunTimeSettings, api); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

func build(args []string, opts *DockerRunOptions) (*models.Job, error) {
	image := args[0]
	parameters := args[1:]
	if err := validateWorkingDir(opts.WorkingDirectory); err != nil {
		return nil, err
	}
	engineSpec, err := models.DockerSpecBuilder(image).
		WithParameters(parameters...).
		WithWorkingDirectory(opts.WorkingDirectory).
		WithEntrypoint(opts.Entrypoint...).
		WithEnvironmentVariables(opts.EnvironmentVariables...).
		Build()
	if err != nil {
		return nil, fmt.Errorf("building docker engine spec: %w", err)
	}

	job, err := helpers.BuildJobFromFlags(engineSpec, opts.JobSettings, opts.TaskSettings)
	if err != nil {
		return nil, fmt.Errorf("building job spec: %w", err)
	}

	// Normalize and validate the job spec
	job.Normalize()
	if err := job.ValidateSubmission(); err != nil {
		return nil, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	return job, nil
}

// Function for validating the workdir of a docker command.
func validateWorkingDir(jobWorkingDir string) error {
	if jobWorkingDir != "" {
		if !strings.HasPrefix(jobWorkingDir, "/") {
			// This mirrors the implementation at path/filepath/path_unix.go#L13 which
			// we reuse here to get cross-platform working dir detection. This is
			// necessary (rather than using IsAbs()) because clients may be running on
			// Windows/Plan9 but we want to check inside Docker (linux).
			return fmt.Errorf("workdir must be an absolute path. Passed in: %s", jobWorkingDir)
		}
	}
	return nil
}
