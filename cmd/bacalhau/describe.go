package bacalhau

import (
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"k8s.io/kubectl/pkg/util/i18n"
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
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolintunparam // incorrectly suggesting unused
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/describe")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		inputJobID := cmdArgs[0]

		j, ok, err := GetAPIClient().Get(ctx, cmdArgs[0])

		if err != nil {
			log.Error().Msgf("Failure retrieving job ID '%s': %s", inputJobID, err)
			return err
		}

		if !ok {
			cmd.Printf("No job ID found matching ID: %s", inputJobID)
			return nil
		}

		shardStates, err := GetAPIClient().GetJobState(ctx, j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job states '%s': %s", j.ID, err)
			return err
		}

		jobEvents, err := GetAPIClient().GetEvents(ctx, j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", j.ID, err)
			return err
		}

		localEvents, err := GetAPIClient().GetLocalEvents(ctx, j.ID)
		if err != nil {
			log.Error().Msgf("Failure retrieving job events '%s': %s", j.ID, err)
			return err
		}

		jobDesc := &model.Job{}
		jobDesc.ID = j.ID
		jobDesc.ClientID = j.ClientID
		jobDesc.RequesterNodeID = j.RequesterNodeID
		jobDesc.Spec = j.Spec
		jobDesc.Deal = j.Deal
		jobDesc.CreatedAt = j.CreatedAt
		jobDesc.State = shardStates

		if OD.IncludeEvents {
			jobDesc.Events = jobEvents
			jobDesc.LocalEvents = localEvents
		}

		const (
			ColumnID        ColumnEnum = "id"
			ColumnCreatedAt ColumnEnum = "created_at"
		)
		bytes, err := yaml.Marshal(jobDesc)
		if err != nil {
			log.Error().Msgf("Failure marshaling job description '%s': %s", j.ID, err)
			return err
		}
		stringDesc := string(bytes)
		cmd.Print(stringDesc)

		return nil
	},
}
