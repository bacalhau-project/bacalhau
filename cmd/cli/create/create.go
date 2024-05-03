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

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/cmd/util/printer"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
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
		bacalhau describe 6e51df50 | bacalhau create `))
)

type CreateOptions struct {
	Filename         string                       // Filename for job (can be .json or .yaml)
	Concurrency      int                          // Number of concurrent jobs to run
	RunTimeSettings  *cliflags.RunTimeSettings    // Run time settings for execution (e.g. wait, get, etc after submission)
	DownloadSettings *cliflags.DownloaderSettings // Settings for running Download
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:         "",
		Concurrency:      1,
		DownloadSettings: cliflags.DefaultDownloaderSettings(),
		RunTimeSettings:  cliflags.DefaultRunTimeSettings(),
	}
}

func NewCmd() *cobra.Command {
	opts := NewCreateOptions()

	cmd := &cobra.Command{
		Use:      "create",
		Short:    "Create a job using a json or yaml file.",
		Long:     createLong,
		Example:  createExample,
		Args:     cobra.MinimumNArgs(0),
		PreRunE:  hook.RemoteCmdPreRunHooks,
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return create(cmd, cmdArgs, opts)
		},
	}

	cliflags.RegisterDownloadFlags(cmd, opts.DownloadSettings)
	cliflags.RegisterRunTimeFlags(cmd, opts.RunTimeSettings)

	return cmd
}

func create(cmd *cobra.Command, cmdArgs []string, OC *CreateOptions) error { //nolint:funlen,gocyclo
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
		return err
	}

	if len(cmdArgs) == 0 {
		byteResult, err = util.ReadFromStdinIfAvailable(cmd)
		if err != nil {
			return fmt.Errorf("unknown error reading from file or stdin: %w", err)
		}
	} else {
		OC.Filename = cmdArgs[0]

		var fileContent *os.File
		fileContent, err = os.Open(OC.Filename)

		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}

		byteResult, err = io.ReadAll(fileContent)
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
	}

	// Do a first pass for parsing to see if it's a Job or JobWithInfo
	err = marshaller.YAMLUnmarshalWithMax(byteResult, &rawMap)
	if err != nil {
		return fmt.Errorf("error parsing file: %w", err)
	}

	// If it's a JobWithInfo, we need to convert it to a Job
	if _, isJobWithInfo := rawMap["Job"]; isJobWithInfo {
		err = marshaller.YAMLUnmarshalWithMax(byteResult, &jwi)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}
		byteResult, err = marshaller.YAMLMarshalWithMax(jwi.Job)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}
	} else if _, isTask := rawMap["with"]; isTask {
		// Else it might be a IPVM Task in JSON format
		var task *model.Task
		task, err = model.UnmarshalIPLD[model.Task](byteResult, json.Decode, model.UCANTaskSchema)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}

		job, err := model.NewJobWithSaneProductionDefaults()
		if err != nil {
			// TODO this is a bit extreme, maybe just ensure the above call doesn't return an error? the mergo package is a bit pointless there.
			panic(err)
		}

		spec, err := task.ToSpec()
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}

		job.Spec = *spec
		byteResult, err = marshaller.YAMLMarshalWithMax(job)
		if err != nil {
			return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
		}
	}

	if len(byteResult) == 0 {
		// TODO better error
		return fmt.Errorf("%s: job is empty", userstrings.JobSpecBad)
	}

	// Turns out the yaml parser supports both yaml & json (because json is a subset of yaml)
	// so we can just use that
	err = marshaller.YAMLUnmarshalWithMax(byteResult, &j)
	if err != nil {
		return fmt.Errorf("%s: %w", userstrings.JobSpecBad, err)
	}

	// See if the job spec is empty
	if j == nil || reflect.DeepEqual(j.Spec, &model.Job{}) {
		// TODO better error
		return fmt.Errorf("%s: job is empty", userstrings.JobSpecBad)
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
			//nolint:staticcheck // TODO: remove this when we have a proper publisher
			Type: j.Spec.Publisher,
		}
	}

	// Warn on fields with data that will be ignored
	if len(unusedFieldList) > 0 {
		cmd.Printf("WARNING: The following fields have data in them and will be ignored on creation: %s\n", strings.Join(unusedFieldList, ", "))
	}

	err = legacy_job.VerifyJob(ctx, j)
	if err != nil {
		return fmt.Errorf("error verifying job: %w", err)
	}
	if OC.RunTimeSettings.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			return fmt.Errorf("error converting job to yaml: %w", err)
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	executingJob, err := util.ExecuteJob(ctx, j)
	if err != nil {
		return fmt.Errorf("error executing job: %w", err)
	}

	err = printer.PrintJobExecutionLegacy(ctx, executingJob, cmd, OC.DownloadSettings, OC.RunTimeSettings, util.GetAPIClient(ctx))
	if err != nil {
		return err
	}

	return nil
}
