package job

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/template"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
)

var (
	runLong = templates.LongDesc(`
		Run a job from a file or from stdin.

		JSON and YAML formats are accepted.
	`)

	runExample = templates.Examples(`
		# Run a job using the data in job.yaml
		bacalhau job run ./job.yaml

		# Run a new job from an already executed job
		bacalhau job describe 6e51df50 | bacalhau job run

		# Download the
		`)
)

type RunOptions struct {
	RunTimeSettings        *cliflags.RunTimeSettings // Run time settings for execution (e.g. follow, wait after submission)
	HideWarnings           bool                      // Show warnings when submitting a job
	NoTemplate             bool
	Force                  bool
	TemplateVars           map[string]string
	TemplateEnvVarsPattern string
}

func NewRunOptions() *RunOptions {
	return &RunOptions{
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
	}
}

func NewRunCmd() *cobra.Command {
	o := NewRunOptions()

	runCmd := &cobra.Command{
		Use:           "run",
		Short:         "Run a job using a json or yaml file.",
		Long:          runLong,
		Example:       runExample,
		Args:          cobra.MinimumNArgs(0),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, args, api)
		},
	}

	runCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(o.RunTimeSettings))
	runCmd.Flags().BoolVar(&o.HideWarnings, "hide-warnings", false, "Hide warnings when submitting a job")
	runCmd.Flags().BoolVar(&o.NoTemplate, "no-template", false,
		"Disable the templating feature. When this flag is set, the job spec will be used as-is, without any placeholder replacements")
	runCmd.Flags().BoolVar(&o.Force, "force", false, "force run even if job spec did not change")
	runCmd.Flags().StringToStringVarP(&o.TemplateVars, "template-vars", "V", nil,
		"Replace a placeholder in the job spec with a value. e.g. --template-vars foo=bar")
	runCmd.Flags().StringVarP(&o.TemplateEnvVarsPattern, "template-envs", "E", "",
		"Specify a regular expression pattern for selecting environment variables to be included as template variables in the job spec."+
			"\ne.g. --template-envs \".*\" will include all environment variables.")

	return runCmd
}

func (o *RunOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()

	// read the job spec from stdin or file
	jobBytes, err := util.ReadJobFromUser(cmd, args)
	if err != nil {
		return err
	}

	if !o.NoTemplate {
		parser, err := template.NewParser(template.ParserParams{
			Replacements: o.TemplateVars,
			EnvPattern:   o.TemplateEnvVarsPattern,
		})
		if err != nil {
			return fmt.Errorf("failed to create template parser: %w", err)
		}
		jobBytes, err = parser.ParseBytes(jobBytes)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}
	}

	j, err := marshaller.UnmarshalJob(jobBytes)
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// Validate the job spec
	err = j.ValidateSubmission()
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	if o.RunTimeSettings.DryRun {
		return o.dryRun(cmd, j, api, ctx)
	}

	// Submit the job
	resp, err := api.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job:   j,
		Force: o.Force,
	})
	if err != nil {
		return fmt.Errorf("failed request: %w", err)
	}

	if !o.HideWarnings && !o.RunTimeSettings.PrintJobIDOnly && len(resp.Warnings) > 0 {
		printer.PrintWarnings(cmd, resp.Warnings)
		cmd.Println()
	}

	j.ID = resp.JobID
	jobProgressPrinter := printer.NewJobProgressPrinter(api, o.RunTimeSettings)
	if err := jobProgressPrinter.PrintJobProgress(ctx, j, cmd); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

func (o *RunOptions) dryRun(
	cmd *cobra.Command,
	newJob *models.Job,
	api client.API,
	ctx context.Context,
) error {
	warnings := newJob.SanitizeSubmission()
	if len(warnings) > 0 {
		printer.PrintWarnings(cmd, warnings)
	}

	diffResponse, diffErr := api.Jobs().Diff(ctx, &apimodels.DiffJobRequest{
		Job: newJob,
	})
	if diffErr != nil {
		printer.PrintError(cmd, fmt.Errorf("failed to fetch Job spec: %w", diffErr))
		printer.PrintWarning(cmd, "falling back to offline mode...")

		outputOps := output.NonTabularOutputOptions{Format: output.YAMLFormat}
		if err := output.OutputOneNonTabular(cmd, outputOps, newJob); err != nil {
			return fmt.Errorf("failed to display local job spec: %w", err)
		}
		return nil
	}

	if diffResponse.Diff == "" {
		printer.PrintWarning(cmd, "No diff detected.")
	} else {
		printer.PrintDiff(cmd, diffResponse.Diff)
	}

	return nil
}
