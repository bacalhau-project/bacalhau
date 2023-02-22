package bacalhau

import (
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
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
)

type DescribeOptions struct {
	Filename      string // Filename for job (can be .json or .yaml)
	IncludeEvents bool   // Include events in the description
	OutputSpec    bool   // Print Just the jobspec to stdout
	JSON          bool   // Print description as JSON
}

func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{
		IncludeEvents: false,
		OutputSpec:    false,
		JSON:          false,
	}
}

func newDescribeCmd() *cobra.Command {
	OD := NewDescribeOptions()

	describeCmd := &cobra.Command{
		Use:     "describe [id]",
		Short:   "Describe a job on the network",
		Long:    describeLong,
		Example: describeExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
			return describe(cmd, cmdArgs, OD)
		},
	}

	describeCmd.PersistentFlags().BoolVar(
		&OD.OutputSpec, "spec", OD.OutputSpec,
		`Output Jobspec to stdout`,
	)
	describeCmd.PersistentFlags().BoolVar(
		&OD.IncludeEvents, "include-events", OD.IncludeEvents,
		`Include events in the description (could be noisy)`,
	)
	describeCmd.PersistentFlags().BoolVar(
		&OD.JSON, "json", OD.JSON,
		`Output description as JSON (if not included will be outputted as YAML by default)`,
	)

	return describeCmd
}

func describe(cmd *cobra.Command, cmdArgs []string, OD *DescribeOptions) error {
	ctx := cmd.Context()

	if err := cmd.ParseFlags(cmdArgs[1:]); err != nil {
		Fatal(cmd, fmt.Sprintf("Failed to parse flags: %v\n", err), 1)
	}

	var err error
	inputJobID := cmdArgs[0]
	if inputJobID == "" {
		var byteResult []byte
		byteResult, err = ReadFromStdinIfAvailable(cmd, cmdArgs)
		// If there's no input ond no stdin, then cmdArgs is nil, and byteResult is nil.
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Unknown error reading from file: %s\n", err), 1)
			return err
		}
		inputJobID = string(byteResult)
	}
	j, foundJob, err := GetAPIClient().Get(ctx, inputJobID)

	if err != nil {
		if er, ok := err.(*bacerrors.ErrorResponse); ok {
			Fatal(cmd, er.Message, 1)
			return nil
		} else {
			Fatal(cmd, fmt.Sprintf("Unknown error trying to get job (ID: %s): %+v", inputJobID, err), 1)
			return nil
		}
	}

	if !foundJob {
		cmd.Printf(err.Error() + "\n")
		Fatal(cmd, "", 1)
	}

	jobDesc := j

	if OD.IncludeEvents {
		jobEvents, innerErr := GetAPIClient().GetEvents(ctx, j.Job.Metadata.ID)
		if innerErr != nil {
			Fatal(cmd, fmt.Sprintf("Failure retrieving job events '%s': %s\n", j.Job.Metadata.ID, innerErr), 1)
		}
		jobDesc.History = jobEvents
	}

	//b, err := model.JSONMarshalIndentWithMax(jobDesc, 3)
	b, err := json.Marshal(jobDesc)
	if err != nil {
		Fatal(cmd, fmt.Sprintf("Failure marshaling job description '%s': %s\n", j.Job.Metadata.ID, err), 1)
	}

	if !OD.JSON {
		// Convert Json to Yaml
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Able to marshal to YAML but not JSON whatttt '%s': %s\n", j.Job.Metadata.ID, err), 1)
		}
		cmd.Print(string(y))
	} else {
		// Print as Json
		cmd.Print(string(b))
	}

	return nil
}
