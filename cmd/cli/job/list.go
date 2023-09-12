package job

import (
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/kubectl/pkg/util/i18n"
)

var orderByFields = []string{"id", "created_at"}

var (
	listShort = `List submitted jobs.`

	listLong = templates.LongDesc(i18n.T(`
		List submitted jobs.
`))

	listExample = templates.Examples(i18n.T(`
		# List submitted jobs.
		bacalhau job list

		# List jobs and output as json
		bacalhau job list --output json --pretty`))

	// defaultLabelFilter is the default label filter for the list command when
	// no other labels are specified.
	defaultLabelFilter = "bacalhau_canary != true"
)

// ListOptions is a struct to support list command
type ListOptions struct {
	output.OutputOptions
	cliflags.ListOptions
	Labels string
}

// NewListOptions returns initialized Options
func NewListOptions() *ListOptions {
	return &ListOptions{
		OutputOptions: output.OutputOptions{Format: output.TableFormat},
		ListOptions: cliflags.ListOptions{
			Limit:         10,
			OrderByFields: orderByFields,
		},
		Labels: defaultLabelFilter,
	}
}

func NewListCmd() *cobra.Command {
	o := NewListOptions()
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   listShort,
		Long:    listLong,
		Example: listExample,
		PreRun:  util.ApplyPorcelainLogLevel,
		Args:    cobra.NoArgs,
		Run:     o.run,
	}

	listCmd.Flags().StringVar(&o.Labels, "labels", o.Labels,
		"Filter nodes by labels. See https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/ for more information.")

	listCmd.Flags().AddFlagSet(cliflags.ListFlags(&o.ListOptions))
	listCmd.Flags().AddFlagSet(cliflags.OutputFormatFlags(&o.OutputOptions))
	return listCmd
}

const (
	listMaxDescWidth = 40
	listMaxDescLines = 10
)

var listColumns = []output.TableColumn[*models.Job]{
	{
		ColumnConfig: table.ColumnConfig{Name: "created", WidthMax: 8, WidthMaxEnforcer: output.ShortenTime},
		Value:        func(j *models.Job) string { return j.GetCreateTime().Format(time.DateTime) },
	},
	{
		ColumnConfig: table.ColumnConfig{
			Name:             "id",
			WidthMax:         idgen.ShortIDLengthWithPrefix,
			WidthMaxEnforcer: func(col string, maxLen int) string { return idgen.ShortID(col) }},
		Value: func(jwi *models.Job) string { return jwi.ID },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "job", WidthMax: listMaxDescWidth, WidthMaxEnforcer: text.WrapText},
		Value: func(j *models.Job) string {
			finalStr := fmt.Sprintf("%v", j.Task().Engine.Type)
			return finalStr[:math.Min(len(finalStr), listMaxDescLines*listMaxDescWidth)]
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "type", WidthMax: 8, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.Job) string { return j.Type },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "state", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value:        func(j *models.Job) string { return j.State.StateType.String() },
	},
}

func (o *ListOptions) run(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()

	var err error
	var labelRequirements []labels.Requirement
	if o.Labels != "" {
		labelRequirements, err = labels.ParseToRequirements(o.Labels)
		if err != nil {
			util.Fatal(cmd, fmt.Errorf("could not parse labels: %w", err), 1)
		}
	}
	response, err := util.GetAPIClientV2(ctx).Jobs().List(&apimodels.ListJobsRequest{
		Labels: labelRequirements,
		BaseListRequest: apimodels.BaseListRequest{
			Limit:     o.Limit,
			NextToken: o.NextToken,
			OrderBy:   o.OrderBy,
			Reverse:   o.Reverse,
		},
	})
	if err != nil {
		util.Fatal(cmd, fmt.Errorf("failed request: %w", err), 1)
	}

	if err = output.Output(cmd, listColumns, o.OutputOptions, response.Jobs); err != nil {
		util.Fatal(cmd, fmt.Errorf("failed to output: %w", err), 1)
	}
}
