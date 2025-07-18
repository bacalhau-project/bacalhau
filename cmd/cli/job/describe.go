package job

import (
	"cmp"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"

	"github.com/bacalhau-project/bacalhau/cmd/util/cols"
	"github.com/bacalhau-project/bacalhau/pkg/lib/collections"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/cmd/util/templates"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

var (
	describeLong = templates.LongDesc(`
		Full description of a job, in yaml format.
		Use 'bacalhau job list' to get a list of jobs.
`)
	describeExample = templates.Examples(`
		# Describe a job with the full ID
		bacalhau job describe j-e3f8c209-d683-4a41-b840-f09b88d087b9

		# Describe a job with the a shortened ID
		bacalhau job describe j-47805f5c

		# Describe a job with json output
		bacalhau job describe --output json --pretty j-b6ad164a
`)
)

// DescribeOptions is a struct to support job command
type DescribeOptions struct {
	OutputOpts output.NonTabularOutputOptions
	JobVersion uint64
	Namespace  string
}

// NewDescribeOptions returns initialized Options
func NewDescribeOptions() *DescribeOptions {
	return &DescribeOptions{
		OutputOpts: output.NonTabularOutputOptions{},
	}
}

func NewDescribeCmd() *cobra.Command {
	o := NewDescribeOptions()
	jobCmd := &cobra.Command{
		Use:           "describe",
		Short:         "Get the info of a job using its name or id.",
		Long:          describeLong,
		Example:       describeExample,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.NewAPIClientManager(cmd, cfg).GetAuthenticatedAPIClient()
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return o.run(cmd, args, api)
		},
	}

	jobCmd.Flags().VarP(util.NewUintValue(0, &o.JobVersion), "version", "v",
		"The job version to filter by. By default, the latest version is used.")
	jobCmd.PersistentFlags().StringVar(&o.Namespace, "namespace", o.Namespace,
		`Job Namespace. If not provided, it will be treated as default namespace.`)
	jobCmd.Flags().AddFlagSet(cliflags.OutputNonTabularFormatFlags(&o.OutputOpts))
	return jobCmd
}

func (o *DescribeOptions) run(cmd *cobra.Command, args []string, api client.API) error {
	ctx := cmd.Context()
	jobIDOrName := args[0]

	request := &apimodels.GetJobRequest{
		JobIDOrName: jobIDOrName,
		JobVersion:  o.JobVersion,
		Include:     "executions,history",
	}
	request.Namespace = o.Namespace
	response, err := api.Jobs().Get(ctx, request)

	if err != nil {
		return err
	}

	if o.OutputOpts.Format != "" {
		if err = output.OutputOneNonTabular(cmd, o.OutputOpts, response); err != nil {
			return fmt.Errorf("failed to write job %s: %w", jobIDOrName, err)
		}
		return nil
	}

	job := response.Job
	var executions []*models.Execution
	if response.Executions != nil {
		// TODO: #520 rename Executions.Executions to Executions.Items
		executions = response.Executions.Items
	}
	// Show most relevant execution first: sort by time DESC
	slices.SortFunc(executions, func(a, b *models.Execution) int {
		return cmp.Compare(b.CreateTime, a.CreateTime)
	})

	var history []*models.JobHistory
	if response.History != nil {
		history = response.History.Items
	}

	o.printHeaderData(cmd, job)
	o.printExecutionsSummary(cmd, executions)

	jobHistory := lo.Filter(history, func(entry *models.JobHistory, _ int) bool {
		return entry.Type == models.JobHistoryTypeJobLevel
	})
	if err = o.printHistory(cmd, "Job", jobHistory); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to write job history: %w", err), 1)
	}

	if err = o.printExecutions(cmd, executions); err != nil {
		return fmt.Errorf("failed to write job executions for job %s: %w", jobIDOrName, err)
	}

	for _, execution := range executions {
		executionHistory := lo.Filter(history, func(item *models.JobHistory, _ int) bool {
			return item.ExecutionID == execution.ID
		})
		if err = o.printHistory(cmd, "Execution "+idgen.ShortUUID(execution.ID), executionHistory); err != nil {
			util.Fatal(cmd, fmt.Errorf("failed to write execution history for %s: %w", execution.ID, err), 1)
		}
	}

	o.printOutputs(cmd, executions)

	return nil
}

func (o *DescribeOptions) printHeaderData(cmd *cobra.Command, job *models.Job) {
	var headerData = []collections.Pair[string, any]{
		{Left: "ID", Right: job.ID},
		{Left: "Name", Right: job.Name},
		{Left: "Namespace", Right: job.Namespace},
		{Left: "Type", Right: job.Type},
		{Left: "State", Right: job.State.StateType},
		{Left: "Message", Right: job.State.Message},
	} // Job type specific data
	if job.Type == models.JobTypeBatch || job.Type == models.JobTypeService {
		headerData = append(headerData, collections.NewPair[string, any]("Count", job.Count))
	}

	// Additional data
	headerData = append(headerData, []collections.Pair[string, any]{
		{Left: "Created Time", Right: job.GetCreateTime().Format(time.DateTime)},
		{Left: "Modified Time", Right: job.GetModifyTime().Format(time.DateTime)},
		{Left: "Version", Right: job.Version},
	}...)

	output.KeyValue(cmd, headerData)
}

func (o *DescribeOptions) printExecutionsSummary(cmd *cobra.Command, executions []*models.Execution) {
	// Summary of executions
	var summaryPairs []collections.Pair[string, any]
	summaryMap := map[models.ExecutionStateType]uint{}
	for _, e := range executions {
		summaryMap[e.ComputeState.StateType]++
	}

	for typ := models.ExecutionStateNew; typ < models.ExecutionStateCancelled; typ++ {
		if summaryMap[typ] > 0 {
			summaryPairs = append(summaryPairs, collections.NewPair[string, any](typ.String(), summaryMap[typ]))
		}
	}
	output.Bold(cmd, "\nSummary\n")
	output.KeyValue(cmd, summaryPairs)
}

func (o *DescribeOptions) printExecutions(cmd *cobra.Command, executions []*models.Execution) error {
	// Executions table
	tableOptions := output.OutputOptions{
		Format:  output.TableFormat,
		NoStyle: true,
	}
	executionCols := []output.TableColumn[*models.Execution]{
		executionColumnID,
		executionColumnNodeID,
		executionColumnState,
		executionColumnDesired,
		executionColumnRev,
		executionColumnCreatedSince,
		executionColumnModifiedSince,
		executionColumnComment,
	}
	output.Bold(cmd, "\nExecutions\n")
	return output.Output(cmd, executionCols, tableOptions, executions)
}

func (o *DescribeOptions) printHistory(cmd *cobra.Command, label string, history []*models.JobHistory) error {
	if len(history) < 1 {
		return nil
	}

	tableOptions := output.OutputOptions{
		Format:  output.TableFormat,
		NoStyle: true,
	}
	jobHistoryCols := []output.TableColumn[*models.JobHistory]{
		cols.HistoryDateTime,
		cols.HistoryTopic,
		cols.HistoryEvent,
	}
	output.Bold(cmd, fmt.Sprintf("\n%s History\n", label))
	return output.Output(cmd, jobHistoryCols, tableOptions, history)
}

func (o *DescribeOptions) printOutputs(cmd *cobra.Command, executions []*models.Execution) {
	outputs := make(map[string]string)
	for _, e := range executions {
		if e.RunOutput != nil {
			separator := ""
			if e.RunOutput.STDOUT != "" {
				outputs[e.ID] = e.RunOutput.STDOUT
				separator = "\n"
			}
			if e.RunOutput.STDERR != "" {
				outputs[e.ID] += separator + e.RunOutput.STDERR
			}
			if e.RunOutput.StdoutTruncated || e.RunOutput.StderrTruncated {
				outputs[e.ID] += "\n...\nOutput truncated"
			}
		}
	}
	if len(outputs) > 0 {
		output.Bold(cmd, "\nStandard Output\n")
		separator := ""
		for id, out := range outputs {
			if len(outputs) == 1 {
				cmd.Print(out)
			} else {
				cmd.Printf("%sExecution %s:\n%s", separator, idgen.ShortUUID(id), out)
			}
			separator = "\n"
		}
	}
}
