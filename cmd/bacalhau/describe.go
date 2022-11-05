package bacalhau

import (
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"
)

var (
	//nolint:lll // Documentation
	describeLong = templates.LongDesc(i18n.T(`
		Full description of a job, in yaml format. Use 'bacalhau list' to get a list of all ids. Short form and long form of the job id are accepted.
`))
	//nolint:lll // Documentation
	describeExample = templates.Examples(i18n.T(`
		# Describe a job with the full ID
		bacalhau describe e3f8c209-d683-4a41-b840-f09b88d087b9

		# Describe a job with the a shortened ID
		bacalhau describe 47805f5c

		# Describe a job and include all server and local events
		bacalhau describe --include-events b6ad164a 
`))

	// Set Defaults (probably a better way to do this)
	OD = NewDescribeOptions()

	// For the -f flag
)

type DescribeOptions struct {
	Filename      string // Filename for job (can be .json or .yaml)
	IncludeEvents bool   // Include events in the description
	OutputSpec    bool   // Print Just the jobspec to stdout
}

func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{
		IncludeEvents: false,
		OutputSpec:    false,
	}
}
func init() { //nolint:gochecknoinits // Using init with Cobra Command is ideomatic
	describeCmd.PersistentFlags().BoolVar(
		&OD.OutputSpec, "spec", OD.OutputSpec,
		`Output Jobspec to stdout`,
	)
	describeCmd.PersistentFlags().BoolVar(
		&OD.IncludeEvents, "include-events", OD.IncludeEvents,
		`Include events in the description (could be noisy)`,
	)
}

var describeCmd = &cobra.Command{
	Use:     "describe [id]",
	Short:   "Describe a job on the network",
	Long:    describeLong,
	Example: describeExample,
	Args:    cobra.ExactArgs(1),
	PreRun:  applyPorcelainLogLevel,
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/describe")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		var err error
		inputJobID := cmdArgs[0]
		if inputJobID == "" {
			var byteResult []byte
			byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
			// If there's no input ond no stdin, then cmdArgs is nil, and byteResult is nil.
			if err != nil {
				Fatal(fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
				return err
			}
			inputJobID = string(byteResult)
		}
		j, foundJob, err := GetAPIClient().Get(ctx, inputJobID)

		if err != nil {
			if er, ok := err.(*bacerrors.ErrorResponse); ok {
				Fatal(er.Message, 1)
				return nil
			} else {
				Fatal(fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", inputJobID, err), 1)
				return nil
			}
		}

		if !foundJob {
			cmd.Printf(err.Error() + "\n")
			Fatal("", 1)
		}

		shardStates, err := GetAPIClient().GetJobState(ctx, j.ID)
		if err != nil {
			Fatal(fmt.Sprintf("Failure retrieving job states '%s': %s\n", j.ID, err), 1)
		}

		jobEvents, err := GetAPIClient().GetEvents(ctx, j.ID)
		if err != nil {
			Fatal(fmt.Sprintf("Failure retrieving job events '%s': %s\n", j.ID, err), 1)
		}

		localEvents, err := GetAPIClient().GetLocalEvents(ctx, j.ID)
		if err != nil {
			Fatal(fmt.Sprintf("Failure retrieving job events '%s': %s\n", j.ID, err), 1)
		}

		jobDesc := j
		jobDesc.State = shardStates

		if OD.IncludeEvents {
			jobDesc.Events = jobEvents
			jobDesc.LocalEvents = localEvents
		}

		const (
			ColumnID        ColumnEnum = "id"
			ColumnCreatedAt ColumnEnum = "created_at"
		)
		b, err := model.JSONMarshalWithMax(jobDesc)
		if err != nil {
			Fatal(fmt.Sprintf("Failure marshaling job description '%s': %s\n", j.ID, err), 1)
		}

		// Convert Json to Yaml
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			Fatal(fmt.Sprintf("Able to marshal to YAML but not JSON whatttt '%s': %s\n", j.ID, err), 1)
		}

		cmd.Print(string(y))

		return nil
	},
}
