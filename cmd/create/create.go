package create

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/flags"
	"github.com/bacalhau-project/bacalhau/cmd/bacalhau/handler"
	"github.com/bacalhau-project/bacalhau/cmd/printer"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/userstrings"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	createLong = templates.LongDesc(i18n.T(`
		Create a job from a file or from stdin.

		JSON and YAML formats are accepted.
	`))
	//nolint:lll // Documentation
	createExample = templates.Examples(i18n.T(`
		# Create a job using the data in job.yaml
		bacalhau create ./job.yaml

		# Create a new job from an already executed job
		bacalhau describe 6e51df50 | bacalhau create -`))
)

type CreateOptions struct {
	Filename        string                    // Filename for job (can be .json or .yaml)
	Concurrency     int                       // Number of concurrent jobs to run
	Confidence      int                       // Minimum number of nodes that must agree on a verification result
	RunTimeSettings *flags.RunTimeSettings    // Run time settings for execution (e.g. wait, get, etc after submission)
	DownloadFlags   *flags.DownloaderSettings // Settings for running Download
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:        "",
		Concurrency:     1,
		Confidence:      0,
		DownloadFlags:   flags.NewDefaultDownloadSettings(),
		RunTimeSettings: flags.NewDefaultRunTimeSettings(),
	}
}

func NewCmd() *cobra.Command {
	OC := NewCreateOptions()

	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a job using a json or yaml file.",
		Long:    createLong,
		Example: createExample,
		Args:    cobra.MinimumNArgs(0),
		//PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			err, exitcode := create(cmd, cmdArgs, OC)
			if err != nil {
				handler.Fatal(cmd, err, exitcode)
			}
			return nil
		},
	}

	createCmd.Flags().AddFlagSet(flags.NewDownloadFlags(OC.DownloadFlags))
	createCmd.Flags().AddFlagSet(flags.NewRunTimeSettingsFlags(OC.RunTimeSettings))

	return createCmd
}

func create(cmd *cobra.Command, cmdArgs []string, OC *CreateOptions) (error, int) { //nolint:funlen,gocyclo
	ctx := cmd.Context()

	// Custom unmarshaller
	// https://stackoverflow.com/questions/70635636/unmarshaling-yaml-into-different-struct-based-off-yaml-field?rq=1
	var jwi model.JobWithInfo
	var j *model.Job
	var err error
	var byteResult []byte
	var rawMap map[string]interface{}

	j, err = model.NewJobWithSaneProductionDefaults()
	if err != nil {
		return err, 1
	}

	if len(cmdArgs) == 0 {
		byteResult, err = handler.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file or stdin: %w", err), handler.ExitError
		}
	} else {
		OC.Filename = cmdArgs[0]

		var fileContent *os.File
		fileContent, err = os.Open(OC.Filename)

		if err != nil {
			return fmt.Errorf("error opening file: %w", err), handler.ExitError
		}

		byteResult, err = io.ReadAll(fileContent)
		if err != nil {
			return fmt.Errorf("error reading file: %w", err), handler.ExitError
		}
	}

	// Do a first pass for parsing to see if it's a Job or JobWithInfo
	err = model.YAMLUnmarshalWithMax(byteResult, &rawMap)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err), handler.ExitError
	}

	// If it's a JobWithInfo, we need to convert it to a Job
	if _, isJobWithInfo := rawMap["Job"]; isJobWithInfo {
		err = model.YAMLUnmarshalWithMax(byteResult, &jwi)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
		}
		byteResult, err = model.YAMLMarshalWithMax(jwi.Job)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
		}
	} else if _, isTask := rawMap["with"]; isTask {
		// Else it might be a IPVM Task in JSON format
		var task *model.Task
		task, err = model.UnmarshalIPLD[model.Task](byteResult, json.Decode, model.UCANTaskSchema)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
		}

		job, err := model.NewJobWithSaneProductionDefaults()
		if err != nil {
			// TODO this is a bit extream, maybe just ensure the above call doesn't return an error? the mergo package is a bit pointless there.
			panic(err)
		}

		spec, err := task.ToSpec()
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
		}

		job.Spec = *spec
		byteResult, err = model.YAMLMarshalWithMax(job)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
		}
	}

	if len(byteResult) == 0 {
		// TODO better error
		return fmt.Errorf("%s", userstrings.JobSpecBad), handler.ExitError
	}

	// Turns out the yaml parser supports both yaml & json (because json is a subset of yaml)
	// so we can just use that
	err = model.YAMLUnmarshalWithMax(byteResult, &j)
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err), handler.ExitError
	}

	// See if the job spec is empty
	if j == nil || reflect.DeepEqual(j.Spec, &model.Job{}) {
		// TODO better error
		return fmt.Errorf("%s", userstrings.JobSpecBad), handler.ExitError
	}

	// Warn on fields with data that will be ignored
	var unusedFieldList []string
	if j.Metadata.ClientID != "" {
		unusedFieldList = append(unusedFieldList, "ClientID")
		j.Metadata.ClientID = ""
	}
	if !reflect.DeepEqual(j.Metadata.CreatedAt, time.Time{}) {
		unusedFieldList = append(unusedFieldList, "CreatedAt")
		j.Metadata.CreatedAt = time.Time{}
	}
	if j.Metadata.ID != "" {
		unusedFieldList = append(unusedFieldList, "ID")
		j.Metadata.ID = ""
	}
	if j.Metadata.Requester.RequesterNodeID != "" {
		unusedFieldList = append(unusedFieldList, "RequesterNodeID")
		j.Metadata.Requester.RequesterNodeID = ""
	}
	if len(j.Metadata.Requester.RequesterPublicKey) != 0 {
		unusedFieldList = append(unusedFieldList, "RequesterPublicKey")
		j.Metadata.Requester.RequesterPublicKey = nil
	}

	if !model.IsValidPublisher(j.Spec.PublisherSpec.Type) {
		j.Spec.PublisherSpec = model.PublisherSpec{
			Type: j.Spec.Publisher,
		}
	}

	// Warn on fields with data that will be ignored
	if len(unusedFieldList) > 0 {
		cmd.Printf("WARNING: The following fields have data in them and will be ignored on creation: %s\n", strings.Join(unusedFieldList, ", "))
	}

	err = jobutils.VerifyJob(ctx, j)
	if err != nil {
		if _, ok := err.(*bacerrors.ImageNotFound); ok {
			return fmt.Errorf("docker image '%s' not found in the registry, or needs authorization", j.Spec.Docker.Image), handler.ExitError
		} else {
			return fmt.Errorf("error verifying job: %w", err), handler.ExitError
		}
	}
	if OC.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			return fmt.Errorf("error converting job to yaml: %w", err), handler.ExitError
		}
		cmd.Print(string(yamlBytes))
		return nil, handler.ExitSuccess
	}

	executingJob, err := handler.ExecuteJob(ctx,
		j,
		*OC.RunTimeSettings,
	)
	if err != nil {
		return fmt.Errorf("error executing job: %w", err), handler.ExitError
	}

	if err := printer.PrintJobExecution(ctx, executingJob, cmd, *OC.DownloadFlags, *OC.RunTimeSettings, handler.GetAPIClient(ctx)); err != nil {
		return err, handler.ExitError
	}

	return nil, handler.ExitSuccess
}
