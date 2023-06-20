package describe

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
	"sigs.k8s.io/yaml"

	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
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

func NewCmd() *cobra.Command {
	OD := NewDescribeOptions()

	describeCmd := &cobra.Command{
		Use:     "describe [id]",
		Short:   "Describe a job on the network",
		Long:    describeLong,
		Example: describeExample,
		Args:    cobra.ExactArgs(1),
		PreRun:  handler.ApplyPorcelainLogLevel,
		Run: func(cmd *cobra.Command, cmdArgs []string) { // nolintunparam // incorrectly suggesting unused
			if err := describe(cmd, cmdArgs, OD); err != nil {
				handler.Fatal(cmd, err, 1)
			}
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
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	var err error
	inputJobID := cmdArgs[0]
	if inputJobID == "" {
		var byteResult []byte
		byteResult, err = handler.ReadFromStdinIfAvailable(cmd)
		// If there's no input ond no stdin, then cmdArgs is nil, and byteResult is nil.
		if err != nil {
			return fmt.Errorf("unknown error reading from file: %w", err)
		}
		inputJobID = string(byteResult)
	}
	j, foundJob, err := handler.GetAPIClient(ctx).Get(ctx, inputJobID)

	if err != nil {
		if err, ok := err.(*bacerrors.ErrorResponse); ok {
			return err
		} else {
			return fmt.Errorf("unknown error trying to get job (ID: %s): %w", inputJobID, err)
		}
	}

	if !foundJob {
		return fmt.Errorf("job not found: %w", err)
	}

	jobDesc := j

	if OD.IncludeEvents {
		jobEvents, err := handler.GetAPIClient(ctx).GetEvents(ctx, j.Job.Metadata.ID, publicapi.EventFilterOptions{})
		if err != nil {
			return fmt.Errorf("failure retrieving job events '%s': %w", j.Job.Metadata.ID, err)
		}
		jobDesc.History = jobEvents
	}

	//b, err := model.JSONMarshalIndentWithMax(jobDesc, 3)
	b, err := json.Marshal(jobDesc)
	if err != nil {
		return fmt.Errorf("failure marshaling job description '%s': %w", j.Job.Metadata.ID, err)
	}

	if !OD.JSON {
		// Convert Json to Yaml
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			return fmt.Errorf("able to marshal to YAML but not JSON whatttt '%s': %w", j.Job.Metadata.ID, err)
		}
		cmd.Print(string(y))
	} else {
		// Print as Json
		cmd.Print(string(b))
	}

	return nil
}
