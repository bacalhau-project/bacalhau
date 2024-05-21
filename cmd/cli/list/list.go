package list

import (
	"errors"
	"fmt"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/cmd/util/flags/cliflags"
	"github.com/bacalhau-project/bacalhau/cmd/util/hook"
	"github.com/bacalhau-project/bacalhau/pkg/lib/math"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/util/templates"
)

var (
	listLong = templates.LongDesc(i18n.T(`
		List jobs on the network.
`))

	listExample = templates.Examples(i18n.T(`
		# List jobs on the network
		bacalhau list

		# List jobs and output as json
		bacalhau list --output json`))

	// The tags that will be excluded by default, if the user does not pass any
	// others to the list command.
	DefaultExcludedTags = []string{
		"canary",
	}
)

type ListOptions struct {
	IDFilter    string               // Filter by Job List to IDs matching substring.
	IncludeTags []string             // Only return jobs with these annotations
	ExcludeTags []string             // Only return jobs without these annotations
	MaxJobs     int                  // Print the first NUM jobs instead of the first 10.
	OutputOpts  output.OutputOptions // The output format for the list of jobs (json or text)
	SortReverse bool                 // Reverse order of table - for time sorting, this will be newest first.
	SortBy      ColumnEnum           // Sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].
	ReturnAll   bool                 // Return all jobs, not just those that belong to the user
}

func NewListOptions() *ListOptions {
	return &ListOptions{
		IDFilter:    "",
		IncludeTags: model.IncludeAny,
		ExcludeTags: DefaultExcludedTags,
		MaxJobs:     10,
		OutputOpts:  output.OutputOptions{Format: output.TableFormat},
		SortReverse: true,
		SortBy:      ColumnCreatedAt,
		ReturnAll:   false,
	}
}

func NewCmd() *cobra.Command {
	OL := NewListOptions()

	listCmd := &cobra.Command{
		Use:      "list",
		Short:    "List jobs on the network",
		Long:     listLong,
		Example:  listExample,
		PreRunE:  hook.RemoteCmdPreRunHooks,
		PostRunE: hook.RemoteCmdPostRunHooks,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// initialize a new or open an existing repo merging any config file(s) it contains into cfg.
			cfg, err := util.SetupRepoConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to setup repo: %w", err)
			}
			// create an api client
			api, err := util.GetAPIClient(cfg)
			if err != nil {
				return fmt.Errorf("failed to create api client: %w", err)
			}
			return list(cmd, api, OL)
		},
	}

	listCmd.PersistentFlags().StringVar(&OL.IDFilter, "id-filter", OL.IDFilter, `filter by Job List to IDs matching substring.`)
	listCmd.PersistentFlags().StringSliceVar(&OL.IncludeTags, "include-tag", OL.IncludeTags,
		`Only return jobs that have the passed tag in their annotations`)
	listCmd.PersistentFlags().StringSliceVar(&OL.ExcludeTags, "exclude-tag", OL.ExcludeTags,
		`Only return jobs that do not have the passed tag in their annotations`)
	listCmd.PersistentFlags().IntVarP(
		&OL.MaxJobs, "number", "n", OL.MaxJobs,
		`print the first NUM jobs instead of the first 10.`,
	)
	listCmd.PersistentFlags().BoolVar(&OL.SortReverse, "reverse", OL.SortReverse,
		//nolint:lll // Documentation
		`reverse order of table - for time sorting, this will be newest first. Use '--reverse=false' to sort oldest first (single quotes are required).`)

	listCmd.PersistentFlags().Var(&OL.SortBy, "sort-by",
		`sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].`)
	listCmd.PersistentFlags().Lookup("sort-by").DefValue = string(ColumnCreatedAt)
	if OL.SortBy == "" {
		OL.SortBy = ColumnCreatedAt
	}
	listCmd.PersistentFlags().BoolVar(
		&OL.ReturnAll, "all", OL.ReturnAll,
		//nolint:lll // Documentation
		`Fetch all jobs from the network (default is to filter those belonging to the user). This option may take a long time to return, please use with caution.`,
	)
	listCmd.PersistentFlags().AddFlagSet(cliflags.OutputFormatFlags(&OL.OutputOpts))

	return listCmd
}

// From: https://stackoverflow.com/questions/50824554/permitted-flag-values-for-cobra
type ColumnEnum string

const (
	ColumnID        ColumnEnum = "id"
	ColumnCreatedAt ColumnEnum = "created_at"
)

func (c *ColumnEnum) String() string {
	return string(*c)
}

// Type is only used in help text
func (c *ColumnEnum) Type() string {
	return "Column"
}

// Set must have pointer receiver so it doesn't change the value of a copy
func (c *ColumnEnum) Set(v string) error {
	switch v {
	case string(ColumnID), string(ColumnCreatedAt):
		*c = ColumnEnum(v)
		return nil
	default:
		return errors.New(`must be one of "id", or "created_at"`)
	}
}

const (
	maxDescWidth = 40
	maxDescLines = 10
)

var listColumns = []output.TableColumn[*model.JobWithInfo]{
	{
		ColumnConfig: table.ColumnConfig{Name: "created", WidthMax: 8, WidthMaxEnforcer: shortenTime},
		Value:        func(j *model.JobWithInfo) string { return j.Job.Metadata.CreatedAt.Format(time.DateTime) },
	},
	{
		ColumnConfig: table.ColumnConfig{
			Name:             "id",
			WidthMax:         idgen.ShortIDLengthWithPrefix,
			WidthMaxEnforcer: func(col string, maxLen int) string { return idgen.ShortUUID(col) }},
		Value: func(jwi *model.JobWithInfo) string { return jwi.Job.ID() },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "job", WidthMax: maxDescWidth, WidthMaxEnforcer: text.WrapText},
		Value: func(j *model.JobWithInfo) string {
			finalStr := fmt.Sprintf("%v", j.Job.Spec.EngineSpec)
			return finalStr[:math.Min(len(finalStr), maxDescLines*maxDescWidth)]
		},
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "state", WidthMax: 20, WidthMaxEnforcer: text.WrapText},
		Value:        func(jwi *model.JobWithInfo) string { return legacy_job.ComputeStateSummary(jwi.State) },
	},
	{
		ColumnConfig: table.ColumnConfig{Name: "published"},
		Value:        legacy_job.ComputeResultsSummary,
	},
}

func list(cmd *cobra.Command, api *client.APIClient, OL *ListOptions) error {
	ctx := cmd.Context()
	jobs, err := api.List(
		ctx,
		OL.IDFilter,
		OL.IncludeTags,
		OL.ExcludeTags,
		OL.MaxJobs,
		OL.ReturnAll,
		OL.SortBy.String(),
		OL.SortReverse,
	)
	if err != nil {
		return err
	}

	return output.Output(cmd, listColumns, OL.OutputOpts, jobs)
}

func shortenTime(formattedTime string, maxLen int) string {
	if len(formattedTime) > maxLen {
		t, err := time.Parse(time.DateTime, formattedTime)
		if err != nil {
			panic(err)
		}
		formattedTime = t.Format(time.TimeOnly)
	}

	return formattedTime
}
