package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	shellescape "gopkg.in/alessio/shellescape.v1"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/parse"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/lib/template"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	getLong = templates.LongDesc(i18n.T(`
		Execute a specific job type.
`))

	//nolint:lll // Documentation
	getExample = templates.Examples(i18n.T(`
		# Execute the app.py script with Python
		bacalhau exec python app.py

		# Run a duckdb query against a CSV file
		bacalhau exec -i src=...,dst=/inputs/data.csv duckdb "select * from /inputs/data.csv"
`))
)

type ExecOptions struct {
	SpecSettings    *cliflags.SpecFlagSettings
	RunTimeSettings *cliflags.RunTimeSettings
	Code            string
}

func NewExecOptions() *ExecOptions {
	return &ExecOptions{
		SpecSettings:    cliflags.NewSpecFlagDefaultSettings(),
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
	}
}

func NewCmd() *cobra.Command {
	options := NewExecOptions()
	return NewCmdWithOptions(options)
}

func NewCmdWithOptions(options *ExecOptions) *cobra.Command {
	execCmd := &cobra.Command{
		Use:                "exec [jobtype]",
		Short:              "Execute a specific job type",
		Long:               getLong,
		Example:            getExample,
		Args:               cobra.MinimumNArgs(1),
		PreRunE:            util.ClientPreRunHooks,
		PostRunE:           util.ClientPostRunHooks,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		Run: func(cmd *cobra.Command, cmdArgs []string) {
			// Find the unknown arguments from the original args.  We only want to find the
			// flags that are unknown. We will only support the long form for custom
			// job types as we will want to use them as keys in template completions.
			unknownArgs := ExtractUnknownArgs(cmd.Flags(), os.Args[1:])

			if err := exec(cmd, cmdArgs, unknownArgs, options); err != nil {
				util.Fatal(cmd, err, 1)
			}
		},
	}

	execCmd.PersistentFlags().AddFlagSet(cliflags.SpecFlags(options.SpecSettings))
	execCmd.PersistentFlags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(options.RunTimeSettings))
	execCmd.Flags().StringVar(&options.Code, "code", "", "Specifies the file, or directory of code to send with the request")

	return execCmd
}

func exec(cmd *cobra.Command, cmdArgs []string, unknownArgs []string, options *ExecOptions) error {
	job, err := PrepareJob(cmd, cmdArgs, unknownArgs, options)
	if err != nil {
		return err
	}

	job.Normalize()
	err = job.ValidateSubmission()
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	client := util.GetAPIClientV2(cmd.Context())
	resp, err := client.Jobs().Put(&apimodels.PutJobRequest{
		Job: job,
	})
	if err != nil {
		return fmt.Errorf("failed request: %w", err)
	}

	if err := printer.PrintJobExecution(cmd.Context(), resp.JobID, cmd, options.RunTimeSettings, client); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

//nolint:funlen
func PrepareJob(cmd *cobra.Command, cmdArgs []string, unknownArgs []string, options *ExecOptions) (*models.Job, error) {
	var err error
	var jobType, templateString string
	var job *models.Job

	// Determine the job type and lookup the template for that type. If we
	// don't have a template, then we don't know how to submit that job type.
	jobType = cmdArgs[0]

	for i := range cmdArgs {
		// If any parameters were quoted, we should make sure we try and add
		// them back in after they were stripped for us.
		if strings.Contains(cmdArgs[i], " ") {
			cmdArgs[i] = shellescape.Quote(cmdArgs[i])
		}
	}

	tpl, err := NewTemplateMap(embeddedFiles, "templates")
	if err != nil {
		return nil, fmt.Errorf("failed to find supported job types, templates missing")
	}

	// Get the template string, or if we can't find one for this type, then
	// provide a list of ones we _do_ support.
	if templateString, err = tpl.Get(jobType); err != nil {
		knownTypes := tpl.AllTemplates()

		supportedTypes := ""
		if len(knownTypes) > 0 {
			supportedTypes = "\nSupported types:\n"

			for _, kt := range knownTypes {
				supportedTypes = supportedTypes + fmt.Sprintf("  * %s\n", kt)
			}
		}

		return nil, fmt.Errorf("the job type '%s' is not supported."+supportedTypes, jobType)
	}

	// Convert the unknown args to a map which we can use to fill in the template
	replacements := flagsToMap(unknownArgs)

	parser, err := template.NewParser(template.ParserParams{
		Replacements: replacements,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create %s job when parsing template: %+w", jobType, err)
	}

	tplResult, err := parser.ParseBytes([]byte(templateString))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// tplResult is now a []byte containing json for the job we will eventually submit.
	if err = json.Unmarshal(tplResult, &job); err != nil {
		return nil, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// Attach the command line arguments that were provided to exec.  These are passed through
	// to the template as Command/Arguments. e.g. `bacalhau exec python app.py` will set
	// Command -> python, and Arguments -> ["app.py"]
	job.Tasks[0].Engine.Params["Command"] = jobType
	job.Tasks[0].Engine.Params["Arguments"] = cmdArgs[1:]

	// Attach any inputs the user specified to the job spec
	if err := prepareInputs(options, job); err != nil {
		return nil, err
	}

	// Process --code if anything was specified. In future we may want to try and determine this
	// ourselves where it is not specified, but it will likely be dependendent on job type.
	if options.Code != "" {
		if err = addInlineContent(cmd.Context(), options.Code, job); err != nil {
			return nil, err
		}
	}

	// Add the default publisher (which is currently IPFS)
	publisherSpec := options.SpecSettings.Publisher.Value()
	job.Tasks[0].Publisher = &models.SpecConfig{
		Type:   publisherSpec.Type.String(),
		Params: publisherSpec.Params,
	}

	// Handle ResultPaths by using the legacy parser and converting.
	if err := prepareJobOutputs(cmd.Context(), options, job); err != nil {
		return nil, err
	}

	// Parse labels from flag, we expect key=value for the non-legacy models.Job
	if err := prepareLabels(options, job); err != nil {
		return nil, err
	}

	// Constraints for node selection
	if err := prepareConstraints(options, job); err != nil {
		return nil, err
	}

	// Environment variables
	if err := prepareEnvVars(options, job); err != nil {
		return nil, err
	}

	// Set the execution timeouts
	job.Tasks[0].Timeouts = &models.TimeoutConfig{
		ExecutionTimeout: options.SpecSettings.Timeout,
	}

	// Unsupported in new job specifications (models.Job)
	// options.SpecSettings.DoNotTrack

	return job, nil
}

func prepareConstraints(options *ExecOptions, job *models.Job) error {
	if nodeSelectorRequirements, err := parse.NodeSelector(options.SpecSettings.Selector); err != nil {
		return err
	} else {
		constraints, err := legacy.FromLegacyLabelSelector(nodeSelectorRequirements)
		if err != nil {
			return err
		}
		job.Constraints = constraints
	}

	return nil
}

func prepareInputs(options *ExecOptions, job *models.Job) error {
	for _, ss := range options.SpecSettings.Inputs.Values() {
		src, err := legacy.FromLegacyStorageSpecToInputSource(ss)
		if err != nil {
			return fmt.Errorf("failed to process input %s: %w", ss.Name, err)
		}

		job.Tasks[0].InputSources = append(job.Tasks[0].InputSources, src)
	}
	return nil
}

func prepareLabels(options *ExecOptions, job *models.Job) error {
	if len(options.SpecSettings.Labels) > 0 {
		if labels, err := parse.StringSliceToMap(options.SpecSettings.Labels); err != nil {
			return err
		} else {
			job.Labels = labels
		}
	}
	return nil
}

func prepareEnvVars(options *ExecOptions, job *models.Job) error {
	if len(options.SpecSettings.EnvVar) > 0 {
		if env, err := parse.StringSliceToMap(options.SpecSettings.EnvVar); err != nil {
			return err
		} else {
			job.Tasks[0].Env = env
		}
	}
	return nil
}

func prepareJobOutputs(ctx context.Context, options *ExecOptions, job *models.Job) error {
	legacyOutputs, err := parse.JobOutputs(ctx, options.SpecSettings.OutputVolumes)
	if err != nil {
		return err
	}

	job.Tasks[0].ResultPaths = make([]*models.ResultPath, 0, len(legacyOutputs))
	for _, output := range legacyOutputs {
		rp := &models.ResultPath{
			Name: output.Name,
			Path: output.Path,
		}

		e := rp.Validate()
		if e != nil {
			return e
		}

		job.Tasks[0].ResultPaths = append(job.Tasks[0].ResultPaths, rp)
	}

	return nil
}

// addInlineContent will use codeLocation to determine if it is a single file or a
// directory and will attach to the job as an inline attachment.
func addInlineContent(ctx context.Context, codeLocation string, job *models.Job) error {
	absPath, err := filepath.Abs(codeLocation)
	if err != nil {
		return err
	}

	target := "/code"

	if finfo, err := os.Stat(absPath); err != nil {
		return fmt.Errorf("file '%s' not found", codeLocation)
	} else {
		if !finfo.IsDir() {
			target = fmt.Sprintf("/code/%s", finfo.Name())
		}
	}

	specConfig, err := inline.NewStorage().Upload(ctx, absPath)
	if err != nil {
		return fmt.Errorf("failed to attach code '%s' to job submission: %w", codeLocation, err)
	}

	job.Tasks[0].InputSources = append(job.Tasks[0].InputSources, &models.InputSource{
		Source: &specConfig,
		Alias:  "code",
		Target: target,
	})

	return nil
}
