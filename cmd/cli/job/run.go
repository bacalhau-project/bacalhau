package job

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/template"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	runLong = templates.LongDesc(i18n.T(`
		Run a job from a file or from stdin.

		JSON and YAML formats are accepted.
	`))
	//nolint:lll // Documentation
	runExample = templates.Examples(i18n.T(`
		# Run a job using the data in job.yaml
		bacalhau job run ./job.yaml

		# Run a new job from an already executed job
		bacalhau job describe 6e51df50 | bacalhau job run
		`))
)

type RunOptions struct {
	RunTimeSettings        *cliflags.RunTimeSettings // Run time settings for execution (e.g. follow, wait after submission)
	ShowWarnings           bool                      // Show warnings when submitting a job
	NoTemplate             bool
	TemplateVars           map[string]string
	TemplateEnvVarsPattern string
}

func NewRunOptions() *RunOptions {
	return &RunOptions{
		//RunTimeSettings: cliflags.DefaultRunTimeSettings(),
	}
}

func NewRunCmd() *cobra.Command {
	o := NewRunOptions()

	runCmd := &cobra.Command{
		Use:     "run",
		Short:   "Run a job using a json or yaml file.",
		Long:    runLong,
		Example: runExample,
		Args:    cobra.MinimumNArgs(0),
		RunE:    o.run,
	}

	//runCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(o.RunTimeSettings))
	runCmd.Flags().BoolVar(&o.ShowWarnings, "show-warnings", false, "Show warnings when submitting a job")
	runCmd.Flags().BoolVar(&o.NoTemplate, "no-template", false,
		"Disable the templating feature. When this flag is set, the job spec will be used as-is, without any placeholder replacements")
	runCmd.Flags().StringToStringVarP(&o.TemplateVars, "template-vars", "V", nil,
		"Replace a placeholder in the job spec with a value. e.g. --template-vars foo=bar")
	runCmd.Flags().StringVarP(&o.TemplateEnvVarsPattern, "template-envs", "E", "",
		"Specify a regular expression pattern for selecting environment variables to be included as template variables in the job spec."+
			"\ne.g. --template-envs \".*\" will include all environment variables.")

	return runCmd
}

func (o *RunOptions) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// read the job spec from stdin or file
	var err error
	var byteResult []byte
	if len(args) == 0 {
		byteResult, err = util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file or stdin: %w", err)
		}
	} else {
		var fileContent *os.File
		fileContent, err = os.Open(args[0])
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}
		defer fileContent.Close()

		byteResult, err = io.ReadAll(fileContent)
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
	}
	if len(byteResult) == 0 {
		return errors.New(userstrings.JobSpecBad)
	}

	if !o.NoTemplate {
		parser, err := template.NewParser(template.ParserParams{
			Replacements: o.TemplateVars,
			EnvPattern:   o.TemplateEnvVarsPattern,
		})
		if err != nil {
			return fmt.Errorf("failed to create template parser: %w", err)
		}
		byteResult, err = parser.ParseBytes(byteResult)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}
	}

	// Turns out the yaml parser supports both yaml & json (because json is a subset of yaml)
	// so we can just use that
	var j *models.Job
	err = marshaller.YAMLUnmarshalWithMax(byteResult, &j)
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// Normalize and validate the job spec
	j.Normalize()
	err = j.ValidateSubmission()
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	if o.RunTimeSettings.DryRun {
		warnings := j.SanitizeSubmission()
		if len(warnings) > 0 {
			o.printWarnings(cmd, warnings)
		}
		outputOps := output.NonTabularOutputOptions{Format: output.YAMLFormat}
		if err = output.OutputOneNonTabular(cmd, outputOps, j); err != nil {
			return fmt.Errorf("failed to write job: %w", err)
		}
		return nil
	}

	// Submit the job
	client := util.GetAPIClientV2(cmd)
	resp, err := client.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: j,
	})
	if err != nil {
		return fmt.Errorf("failed request: %w", err)
	}

	if o.ShowWarnings && len(resp.Warnings) > 0 {
		o.printWarnings(cmd, resp.Warnings)
	}

	if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, o.RunTimeSettings, client); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

func (o *RunOptions) printWarnings(cmd *cobra.Command, warnings []string) {
	cmd.Println("Warnings:")
	for _, warning := range warnings {
		cmd.Printf("\t* %s\n", warning)
	}
}
