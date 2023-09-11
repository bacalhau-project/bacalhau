package job

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

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
		`))
)

type RunOptions struct {
	RunTimeSettings *cliflags.RunTimeSettings // Run time settings for execution (e.g. follow, wait after submission)
	ShowWarnings    bool                      // Show warnings when submitting a job
}

func NewRunOptions() *RunOptions {
	return &RunOptions{
		RunTimeSettings: cliflags.DefaultRunTimeSettings(),
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
		PreRun:  util.ApplyPorcelainLogLevel,
		Run:     o.run,
	}

	runCmd.Flags().AddFlagSet(cliflags.NewRunTimeSettingsFlags(o.RunTimeSettings))
	runCmd.Flags().BoolVar(&o.ShowWarnings, "show-warnings", false, "Show warnings when submitting a job")
	return runCmd
}

func (o *RunOptions) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()

	// read the job spec from stdin or file
	var err error
	var byteResult []byte
	if len(args) == 0 {
		byteResult, err = util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			util.Fatal(cmd, fmt.Errorf("unknown error reading from file or stdin: %w", err), 1)
		}
	} else {
		var fileContent *os.File
		fileContent, err = os.Open(args[0])
		if err != nil {
			util.Fatal(cmd, fmt.Errorf("error opening file: %w", err), 1)
		}
		defer fileContent.Close()

		byteResult, err = io.ReadAll(fileContent)
		if err != nil {
			util.Fatal(cmd, fmt.Errorf("error reading file: %w", err), 1)
		}
	}
	if len(byteResult) == 0 {
		util.Fatal(cmd, errors.New(userstrings.JobSpecBad), 1)
	}

	// Turns out the yaml parser supports both yaml & json (because json is a subset of yaml)
	// so we can just use that
	var j *models.Job
	err = marshaller.YAMLUnmarshalWithMax(byteResult, &j)
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), 1)
	}

	// Normalize and validate the job spec
	j.Normalize()
	err = j.ValidateSubmission()
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), 1)
	}

	if o.RunTimeSettings.DryRun {
		warnings := j.SanitizeSubmission()
		if len(warnings) > 0 {
			o.printWarnings(cmd, warnings)
		}
		return
	}

	// Submit the job
	client := util.GetAPIClientV2(ctx)
	resp, err := client.Jobs().Put(&apimodels.PutJobRequest{
		Job: j,
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("failed request: %w", err), 1)
	}

	if o.ShowWarnings && len(resp.Warnings) > 0 {
		o.printWarnings(cmd, resp.Warnings)
	}

	if err := printer.PrintJobExecution(ctx, resp.JobID, cmd, o.RunTimeSettings, client); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to print job execution: %w", err), 1)
	}
}

func (o *RunOptions) printWarnings(cmd *cobra.Command, warnings []string) {
	cmd.Println("Warnings:")
	for _, warning := range warnings {
		cmd.Printf("\t* %s\n", warning)
	}
}
