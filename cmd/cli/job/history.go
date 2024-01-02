package job

import (
	"fmt"
	"strconv"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	historyShort = `List history events for a job by id.`

	historyLong = templates.LongDesc(i18n.T(`
		List job history events for a job by id.
`))

	historyExample = templates.Examples(i18n.T(`
		# All events for a given job.
		bacalhau job history e3f8c209-d683-4a41-b840-f09b88d087b9

		# Job level events
		bacalhau job history --type job e3f8c209

		# Execution level events
		bacalhau job history --type execution e3f8c209
`))
)

// HistoryOptions is a struct to support node command
type HistoryOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	EventType   string
	ExecutionID string
	NodeID      string
}

// NewHistoryOptions returns initialized Options
func NewHistoryOptions() *HistoryOptions {
	return &HistoryOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		EventType:     "all",
	}
}

func NewHistoryCmd() *cobra.Command {
	o := NewHistoryOptions()
	nodeCmd := &cobra.Command{
		Use:     "history [id]",
		Short:   historyShort,
		Long:    historyLong,
		Example: historyExample,
		Args:    cobra.ExactArgs(1),
		Run:     o.run,
	}

	nodeCmd.Flags().StringVar(&o.EventType, "event-type", o.EventType,
		"The type of history events to return. One of: all, job, execution")
	nodeCmd.Flags().StringVar(&o.ExecutionID, "execution-id", o.ExecutionID,
		"The execution id to filter by.")
	nodeCmd.Flags().StringVar(&o.NodeID, "node-id", o.NodeID,
		"The node id to filter by.")
	nodeCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	nodeCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return nodeCmd
}

var historyColumns = []output.TableColumn[*models.JobHistory]{
	{
		ColumnConfig: table.ColumnConfig{Name: "Time", WidthMax: 8, WidthMaxEnforcer: output.ShortenTime},
		Value:        func(j *models.JobHistory) string { return j.Time.Format(time.DateTime) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Level", WidthMax: 15, WidthMaxEnforcer: text.WrapText},
		Value:        func(jwi *models.JobHistory) string { return jwi.Type.String() },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Exec. ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return idgen.ShortID(j.ExecutionID) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Node ID", WidthMax: 10, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return idgen.ShortID(j.NodeID) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Rev.", WidthMax: 4, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return strconv.FormatUint(j.NewRevision, 10) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "Previous State", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string {
			if j.Type == models.JobHistoryTypeJobLevel {
				return j.JobState.Previous.String()
			}
			return j.ExecutionState.Previous.String()
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "New State", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.JobHistory) string {
			if j.Type == models.JobHistoryTypeJobLevel {
				return j.JobState.New.String()
			}
			return j.ExecutionState.New.String()
		},
	},

	{
		ColumnConfig: table.ColumnConfig{Name: "Comment", WidthMax: 40, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.JobHistory) string { return j.Comment },
	},
}

func (o *HistoryOptions) run(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	jobID := args[0]
	response, err := util.GetAPIClientV2(ctx).Jobs().History(&apimodels.ListJobHistoryRequest{
		JobID:       jobID,
		EventType:   o.EventType,
		ExecutionID: o.ExecutionID,
		NodeID:      o.NodeID,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		util.Fatal(cmd, err, 1)
	}

	if err = output.Output(cmd, historyColumns, o.OutputOptions, response.History); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to output: %w", err), 1)
	}
}
