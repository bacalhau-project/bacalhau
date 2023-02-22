package bacalhau

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/downloader/util"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/userstrings"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/ipld/go-ipld-prime/codec/json"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"
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
	Filename        string                   // Filename for job (can be .json or .yaml)
	Concurrency     int                      // Number of concurrent jobs to run
	Confidence      int                      // Minimum number of nodes that must agree on a verification result
	RunTimeSettings RunTimeSettings          // Run time settings for execution (e.g. wait, get, etc after submission)
	DownloadFlags   model.DownloaderSettings // Settings for running Download
	DryRun          bool
}

func NewCreateOptions() *CreateOptions {
	return &CreateOptions{
		Filename:        "",
		Concurrency:     1,
		Confidence:      0,
		DownloadFlags:   *util.NewDownloadSettings(),
		RunTimeSettings: *NewRunTimeSettings(),
	}
}

func newCreateCmd() *cobra.Command {
	OC := NewCreateOptions()

	createCmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a job using a json or yaml file.",
		Long:    createLong,
		Example: createExample,
		Args:    cobra.MinimumNArgs(0),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error {
			return create(cmd, cmdArgs, OC)
		},
	}

	createCmd.Flags().AddFlagSet(NewIPFSDownloadFlags(&OC.DownloadFlags))
	createCmd.Flags().AddFlagSet(NewRunTimeSettingsFlags(&OC.RunTimeSettings))
	createCmd.PersistentFlags().BoolVar(
		&OC.DryRun, "dry-run", OC.DryRun,
		`Do not submit the job, but instead print out what will be submitted`,
	)

	return createCmd
}

func create(cmd *cobra.Command, cmdArgs []string, OC *CreateOptions) error { //nolint:funlen,gocyclo
	ctx := cmd.Context()

	cm := cmd.Context().Value(systemManagerKey).(*system.CleanupManager)

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
		byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Unknown error reading from file or stdin: %s\n", err), 1)
			return err
		}
	} else {
		OC.Filename = cmdArgs[0]

		var fileContent *os.File
		fileContent, err = os.Open(OC.Filename)

		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error opening file: %s", err), 1)
			return err
		}

		byteResult, err = io.ReadAll(fileContent)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error reading file: %s", err), 1)
			return err
		}
	}

	// Do a first pass for parsing to see if it's a Job or JobWithInfo
	err = model.YAMLUnmarshalWithMax(byteResult, &rawMap)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error parsing file: %s", err), 1)
		return err
	}

	// If it's a JobWithInfo, we need to convert it to a Job
	if _, isJobWithInfo := rawMap["Job"]; isJobWithInfo {
		err = model.YAMLUnmarshalWithMax(byteResult, &jwi)
		if err != nil {
			Fatal(cmd, userstrings.JobSpecBad, 1)
			return err
		}
		byteResult, err = model.YAMLMarshalWithMax(jwi.Job)
		if err != nil {
			Fatal(cmd, userstrings.JobSpecBad, 1)
			return err
		}
	} else if _, isTask := rawMap["with"]; isTask {
		// Else it might be a IPVM Task in JSON format
		var task *model.Task
		task, taskErr := model.UnmarshalIPLD[model.Task](byteResult, json.Decode, model.UCANTaskSchema)
		if taskErr != nil {
			Fatal(cmd, userstrings.JobSpecBad, 1)
			return taskErr
		}

		job, taskErr := model.NewJobWithSaneProductionDefaults()
		if taskErr != nil {
			panic(taskErr)
		}

		spec, taskErr := task.ToSpec()
		if taskErr != nil {
			Fatal(cmd, userstrings.JobSpecBad, 1)
			return taskErr
		}

		job.Spec = *spec
		byteResult, taskErr = model.YAMLMarshalWithMax(job)
		if taskErr != nil {
			Fatal(cmd, userstrings.JobSpecBad, 1)
			return taskErr
		}
	}

	if len(byteResult) == 0 {
		Fatal(cmd, userstrings.JobSpecBad, 1)
		return err
	}

	// Turns out the yaml parser supports both yaml & json (because json is a subset of yaml)
	// so we can just use that
	err = model.YAMLUnmarshalWithMax(byteResult, &j)
	if err != nil {
		Fatal(cmd, userstrings.JobSpecBad, 1)
		return err
	}

	// See if the job spec is empty
	if j == nil || reflect.DeepEqual(j.Spec, &model.Job{}) {
		Fatal(cmd, userstrings.JobSpecBad, 1)
		return err
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
	if !reflect.DeepEqual(j.Spec.ExecutionPlan, model.JobExecutionPlan{}) {
		unusedFieldList = append(unusedFieldList, "Verification")
		j.Spec.ExecutionPlan = model.JobExecutionPlan{}
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

	// Warn on fields with data that will be ignored
	if len(unusedFieldList) > 0 {
		cmd.Printf("WARNING: The following fields have data in them and will be ignored on creation: %s\n", strings.Join(unusedFieldList, ", "))
	}

	err = jobutils.VerifyJob(ctx, j)
	if err != nil {
		if _, ok := err.(*bacerrors.ImageNotFound); ok {
			Fatal(cmd, fmt.Sprintf("Docker image '%s' not found in the registry, or needs authorization.", j.Spec.Docker.Image), 1)
			return err
		} else {
			Fatal(cmd, fmt.Sprintf("Error verifying job: %s", err), 1)
			return err
		}
	}
	if OC.DryRun {
		// Converting job to yaml
		var yamlBytes []byte
		yamlBytes, err = yaml.Marshal(j)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error converting job to yaml: %s", err), 1)
			return err
		}
		cmd.Print(string(yamlBytes))
		return nil
	}

	err = ExecuteJob(ctx,
		cm,
		cmd,
		j,
		OC.RunTimeSettings,
		OC.DownloadFlags,
	)

	if err != nil {
		Fatal(cmd, fmt.Sprintf("Error executing job: %s", err), 1)
		return err
	}

	return nil
}
