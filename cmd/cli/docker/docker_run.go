package docker

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/kubectl/pkg/util/i18n"

	"k8s.io/kubectl/pkg/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/cli/helpers"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/configflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	engine_docker "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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

	JobSettings     *cliflags.JobSettings
	TaskSettings    *cliflags.TaskSettings
	RunTimeSettings *cliflags.RunTimeSettings
}

func NewDockerRunOptions() *DockerRunOptions {
	return &DockerRunOptions{
		Entrypoint:       nil,
		WorkingDirectory: "",

		JobSettings:     cliflags.DefaultJobSettings(),
		TaskSettings:    cliflags.DefaultTaskSettings(),
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
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
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create v2 api client: %w", err)
			}
			return run(cmd, cmdArgs, api, opts)
		},
	}

	cliflags.RegisterJobFlags(dockerRunCmd, opts.JobSettings)
	cliflags.RegisterTaskFlags(dockerRunCmd, opts.TaskSettings)
	dockerRunCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(opts.RunTimeSettings))

	if err := configflags.RegisterFlags(dockerRunCmd, dockerRunFlags); err != nil {
		util.Fatal(dockerRunCmd, err, 1)
	}
	// register flags unique to docker.
	dockerFlags := pflag.NewFlagSet("docker", pflag.ContinueOnError)
	dockerFlags.StringVarP(&opts.WorkingDirectory, "workdir", "w", opts.WorkingDirectory,
		`Working directory inside the container. Overrides the working directory shipped with the image (e.g. via WORKDIR in Dockerfile).`)
	dockerFlags.StringSliceVar(&opts.Entrypoint, "entrypoint", opts.Entrypoint,
		`Override the default ENTRYPOINT of the image`)
	dockerFlags.StringSliceVarP(&opts.EnvironmentVariables, "env", "e", opts.EnvironmentVariables,
		"The environment variables to supply to the job (e.g. --env FOO=bar --env BAR=baz)")

	dockerRunCmd.Flags().AddFlagSet(dockerFlags)

	return dockerRunCmd
}

func run(cmd *cobra.Command, args []string, api clientv2.API, opts *DockerRunOptions) error {
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

	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	if err != nil {
		return bacerrors.Wrap(err, "failed to submit job")
	}

	if len(resp.Warnings) > 0 {
		helpers.PrintWarnings(cmd, resp.Warnings)
	}

	job.ID = resp.JobID
	jobProgressPrinter := printer.NewJobProgressPrinter(api, opts.RunTimeSettings)
	if err := jobProgressPrinter.PrintJobProgress(ctx, job, cmd); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

func build(args []string, opts *DockerRunOptions) (*models.Job, error) {
	image := args[0]
	parameters := args[1:]
	engineSpec, err := engine_docker.NewDockerEngineBuilder(image).
		WithParameters(parameters...).
		WithWorkingDirectory(opts.WorkingDirectory).
		WithEntrypoint(opts.Entrypoint...).
		WithEnvironmentVariables(opts.EnvironmentVariables...).
		Build()
	if err != nil {
		return nil, err
	}

	return helpers.BuildJobFromFlags(engineSpec, opts.JobSettings, opts.TaskSettings)
}
