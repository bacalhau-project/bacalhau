package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/alessio/shellescape.v1"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	"github.com/bacalhau-project/bacalhau/pkg/lib/template"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	getLong = templates.LongDesc(i18n.T(
		fmt.Sprintf(`Execute a specific job type.

Allows for the execution of a job type with the given code,
without the need to create a container, or webassembly module.
By specifying the code with the '--code' flag you can ship the code
to the cluster for execution, specified by the remainder of the
command line.  See examples below.

Supported job types:

%s
		`, supportedJobTypes()),
	))

	//nolint:lll // Documentation
	getExample = templates.Examples(i18n.T(`
		# Execute the app.py script with Python
		bacalhau exec --code app.py python app.py

		# Run a duckdb query against a CSV file
		bacalhau exec -i src=...,dst=/inputs/data.csv duckdb "select * from /inputs/data.csv"
`))
)

type ExecOptions struct {
	JobSettings     *cliflags.JobSettings
	TaskSettings    *cliflags.TaskSettings
	RunTimeSettings *cliflags.RunTimeSettings
	Code            string
}

func NewExecOptions() *ExecOptions {
	return &ExecOptions{
		JobSettings:     cliflags.DefaultJobSettings(),
		TaskSettings:    cliflags.DefaultTaskSettings(),
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
		PreRunE:            hook.RemoteCmdPreRunHooks,
		PostRunE:           hook.RemoteCmdPostRunHooks,
		FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			// Find the unknown arguments from the original args.  We only want to find the
			// flags that are unknown. We will only support the long form for custom
			// job types as we will want to use them as keys in template completions.
			unknownArgs := ExtractUnknownArgs(cmd.Flags(), os.Args[1:])
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClientV2(cmd, cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return exec(cmd, cmdArgs, unknownArgs, api, options)
		},
	}

	cliflags.RegisterJobFlags(execCmd, options.JobSettings)
	cliflags.RegisterTaskFlags(execCmd, options.TaskSettings)

	execCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(options.RunTimeSettings))
	execCmd.Flags().StringVar(&options.Code, "code", "", "Specifies the file, or directory of code to send with the request")

	return execCmd
}

func exec(cmd *cobra.Command, cmdArgs []string, unknownArgs []string, api client.API, options *ExecOptions) error {
	job, err := PrepareJob(cmd, cmdArgs, unknownArgs, options)
	if err != nil {
		return err
	}

	job.Normalize()
	err = job.ValidateSubmission()
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	resp, err := api.Jobs().Put(cmd.Context(), &apimodels.PutJobRequest{
		Job: job,
	})
	if err != nil {
		return fmt.Errorf("failed request: %w", err)
	}

	job.ID = resp.JobID
	jobProgressPrinter := printer.NewJobProgressPrinter(api, options.RunTimeSettings)
	if err := jobProgressPrinter.PrintJobProgress(cmd.Context(), job, cmd); err != nil {
		return fmt.Errorf("failed to print job execution: %w", err)
	}

	return nil
}

// Provides a string to diplay the currently available job types
func supportedJobTypes() string {
	tpl, _ := NewTemplateMap(embeddedFiles, "templates")
	var sb strings.Builder
	for _, s := range tpl.AllTemplates() {
		sb.WriteString(fmt.Sprintf("  * %s\n", s))
	}
	return sb.String()
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

	// Process --code if anything was specified. In future we may want to try and determine this
	// ourselves where it is not specified, but it will likely be dependent on job type.
	if options.Code != "" {
		if err = addInlineContent(cmd.Context(), options.Code, job); err != nil {
			return nil, err
		}
	}

	job.Labels, err = options.JobSettings.Labels()
	job.Task().Publisher = options.TaskSettings.Publisher.Value()
	job.Task().ResultPaths = options.TaskSettings.ResultPaths
	job.Task().Env = options.TaskSettings.EnvironmentVariables
	job.Task().InputSources = options.TaskSettings.InputSources.Values()
	if err != nil {
		return nil, fmt.Errorf("parsing job labels: %w", err)
	}
	job.Constraints, err = options.JobSettings.Constraints()
	if err != nil {
		return nil, fmt.Errorf("parsing job constraints: %w", err)
	}

	// Set the execution timeouts
	job.Tasks[0].Timeouts = &models.TimeoutConfig{
		TotalTimeout: options.TaskSettings.Timeout,
	}

	return job, nil
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
