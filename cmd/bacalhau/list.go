package bacalhau

import (
	"errors"
	"fmt"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/templates"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"k8s.io/kubectl/pkg/util/i18n"
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
	defaultExcludedTags = []model.ExcludedTag{
		"canary",
	}
)

type ListOptions struct {
	HideHeader   bool                // Hide the column headers
	IDFilter     string              // Filter by Job List to IDs matching substring.
	IncludeTags  []model.IncludedTag // Only return jobs with these annotations
	ExcludeTags  []model.ExcludedTag // Only return jobs without these annotations
	NoStyle      bool                // Remove all styling from table output.
	MaxJobs      int                 // Print the first NUM jobs instead of the first 10.
	OutputFormat string              // The output format for the list of jobs (json or text)
	SortReverse  bool                // Reverse order of table - for time sorting, this will be newest first.
	SortBy       ColumnEnum          // Sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].
	OutputWide   bool                // Print full values in the table results
	ReturnAll    bool                // Return all jobs, not just those that belong to the user
}

func NewListOptions() *ListOptions {
	return &ListOptions{
		HideHeader:   false,
		IDFilter:     "",
		IncludeTags:  model.IncludeAny,
		ExcludeTags:  defaultExcludedTags,
		NoStyle:      false,
		MaxJobs:      10,
		OutputFormat: "text",
		SortReverse:  true,
		SortBy:       ColumnCreatedAt,
		OutputWide:   false,
		ReturnAll:    false,
	}
}

func newListCmd() *cobra.Command {
	OL := NewListOptions()

	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List jobs on the network",
		Long:    listLong,
		Example: listExample,
		PreRun:  applyPorcelainLogLevel,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return list(cmd, OL)
		},
	}

	listCmd.PersistentFlags().BoolVar(&OL.HideHeader, "hide-header", OL.HideHeader,
		`do not print the column headers.`)
	listCmd.PersistentFlags().StringVar(&OL.IDFilter, "id-filter", OL.IDFilter, `filter by Job List to IDs matching substring.`)
	listCmd.PersistentFlags().Var(IncludedTagFlag(&OL.IncludeTags), "include-tag",
		`Only return jobs that have the passed tag in their annotations`)
	listCmd.PersistentFlags().Var(ExcludedTagFlag(&OL.ExcludeTags), "exclude-tag",
		`Only return jobs that do not have the passed tag in their annotations`)
	listCmd.PersistentFlags().BoolVar(&OL.NoStyle, "no-style", OL.NoStyle, `remove all styling from table output.`)
	listCmd.PersistentFlags().IntVarP(
		&OL.MaxJobs, "number", "n", OL.MaxJobs,
		`print the first NUM jobs instead of the first 10.`,
	)
	listCmd.PersistentFlags().StringVar(
		&OL.OutputFormat, "output", OL.OutputFormat,
		`The output format for the list of jobs (json or text)`,
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
		&OL.OutputWide, "wide", OL.OutputWide,
		`Print full values in the table results`,
	)
	listCmd.PersistentFlags().BoolVar(
		&OL.ReturnAll, "all", OL.ReturnAll,
		//nolint:lll // Documentation
		`Fetch all jobs from the network (default is to filter those belonging to the user). This option may take a long time to return, please use with caution.`,
	)

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

func list(cmd *cobra.Command, OL *ListOptions) error {
	ctx := cmd.Context()

	log.Ctx(ctx).Debug().Msgf("Table filter flag set to: %s", OL.IDFilter)
	log.Ctx(ctx).Debug().Msgf("Table limit flag set to: %d", OL.MaxJobs)
	log.Ctx(ctx).Debug().Msgf("Table output format flag set to: %s", OL.OutputFormat)
	log.Ctx(ctx).Debug().Msgf("Table reverse flag set to: %t", OL.SortReverse)
	log.Ctx(ctx).Debug().Msgf("Found return all flag: %t", OL.ReturnAll)
	log.Ctx(ctx).Debug().Msgf("Found sort flag: %s", OL.SortBy)
	log.Ctx(ctx).Debug().Msgf("Found hide header flag set to: %t", OL.HideHeader)
	log.Ctx(ctx).Debug().Msgf("Found no-style header flag set to: %t", OL.NoStyle)
	log.Ctx(ctx).Debug().Msgf("Found output wide flag set to: %t", OL.OutputWide)

	jobs, err := GetAPIClient().List(
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
		Fatal(cmd, fmt.Sprintf("Error listing jobs: %s", err), 1)
	}

	numberInTable := system.Min(OL.MaxJobs, len(jobs))
	log.Ctx(ctx).Debug().Msgf("Number of jobs printing: %d", numberInTable)

	var msgBytes []byte
	if OL.OutputFormat == JSONFormat {
		msgBytes, err = model.JSONMarshalWithMax(jobs)
		if err != nil {
			Fatal(cmd, fmt.Sprintf("Error marshaling jobs to JSON: %s", err), 1)
		}
		cmd.Printf("%s\n", msgBytes)
	} else {
		tw := table.NewWriter()
		tw.SetOutputMirror(cmd.OutOrStderr())
		if !OL.HideHeader {
			tw.AppendHeader(table.Row{"created", "id", "job", "state", "verified", "published"})
		}
		columnConfig := []table.ColumnConfig{}
		tw.SetColumnConfigs(columnConfig)

		var rows []table.Row
		for _, j := range jobs {
			var summaryRow table.Row
			summaryRow, err = summarizeJob(j, OL)
			if err != nil {
				Fatal(cmd, fmt.Sprintf("Error summarizing job: %s", err), 1)
			}
			rows = append(rows, summaryRow)
		}
		if err != nil {
			return err
		}
		tw.AppendRows(rows)

		if OL.NoStyle {
			tw.SetStyle(table.Style{
				Name:   "StyleDefault",
				Box:    table.StyleBoxDefault,
				Color:  table.ColorOptionsDefault,
				Format: table.FormatOptionsDefault,
				HTML:   table.DefaultHTMLOptions,
				Options: table.Options{
					DrawBorder:      false,
					SeparateColumns: false,
					SeparateFooter:  false,
					SeparateHeader:  false,
					SeparateRows:    false,
				},
				Title: table.TitleOptionsDefault,
			})
		} else {
			tw.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		}

		tw.Render()
	}

	return nil
}

// Renders job details into a table row
func summarizeJob(j *model.JobWithInfo, OL *ListOptions) (table.Row, error) {
	jobDesc := []string{
		j.Job.Spec.Engine.String(),
	}
	// Add more details to the job description (e.g. Docker ubuntu echo Hello World)
	if j.Job.Spec.Engine == model.EngineDocker {
		jobDesc = append(jobDesc, j.Job.Spec.Docker.Image, strings.Join(j.Job.Spec.Docker.Entrypoint, " "))
	}

	// compute state summary
	//nolint:gocritic
	stateSummary := job.ComputeStateSummary(j.State)

	// compute verifiedSummary
	verifiedSummary := job.ComputeVerifiedSummary(j)

	// compute resultSummary
	resultSummary := job.ComputeResultsSummary(j)

	row := table.Row{
		shortenTime(OL.OutputWide, j.Job.Metadata.CreatedAt),
		shortID(OL.OutputWide, j.Job.Metadata.ID),
		shortenString(OL.OutputWide, strings.Join(jobDesc, " ")),
		shortenString(OL.OutputWide, stateSummary),
		shortenString(OL.OutputWide, verifiedSummary),
		shortenString(OL.OutputWide, resultSummary),
	}

	return row, nil
}
