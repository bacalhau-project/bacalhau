package bacalhau

import (
	"context"
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

	//nolint:lll // Documentation
	listExample = templates.Examples(i18n.T(`
		# List jobs on the network
		bacalhau list

		# List jobs and output as json
		bacalhau list --output json`))

	// Set Defaults (probably a better way to do this)
	OL = NewListOptions()

	// For the -f flag
)

type ListOptions struct {
	HideHeader   bool       // Hide the column headers
	IDFilter     string     // Filter by Job List to IDs matching substring.
	NoStyle      bool       // Remove all styling from table output.
	MaxJobs      int        // Print the first NUM jobs instead of the first 10.
	OutputFormat string     // The output format for the list of jobs (json or text)
	SortReverse  bool       // Reverse order of table - for time sorting, this will be newest first.
	SortBy       ColumnEnum // Sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].
	OutputWide   bool       // Print full values in the table results
	ReturnAll    bool       // Return all jobs, not just those that belong to the user
}

func NewListOptions() *ListOptions {
	return &ListOptions{
		HideHeader:   false,
		IDFilter:     "",
		NoStyle:      false,
		MaxJobs:      10,
		OutputFormat: "text",
		SortReverse:  true,
		SortBy:       ColumnCreatedAt,
		OutputWide:   false,
		ReturnAll:    false,
	}
}

func init() { //nolint:gochecknoinits // Using init in cobra command is idomatic
	listCmd.PersistentFlags().BoolVar(&OL.HideHeader, "hide-header", OL.HideHeader,
		`do not print the column headers.`)
	listCmd.PersistentFlags().StringVar(&OL.IDFilter, "id-filter", OL.IDFilter, `filter by Job List to IDs matching substring.`)
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

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List jobs on the network",
	Long:    listLong,
	Example: listExample,
	PreRun:  applyPorcelainLogLevel,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		ctx, rootSpan := system.NewRootSpan(ctx, system.GetTracer(), "cmd/bacalhau/list")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		log.Debug().Msgf("Table filter flag set to: %s", OL.IDFilter)
		log.Debug().Msgf("Table limit flag set to: %d", OL.MaxJobs)
		log.Debug().Msgf("Table output format flag set to: %s", OL.OutputFormat)
		log.Debug().Msgf("Table reverse flag set to: %t", OL.SortReverse)
		log.Debug().Msgf("Found return all flag: %t", OL.ReturnAll)
		log.Debug().Msgf("Found sort flag: %s", OL.SortBy)
		log.Debug().Msgf("Found hide header flag set to: %t", OL.HideHeader)
		log.Debug().Msgf("Found no-style header flag set to: %t", OL.NoStyle)
		log.Debug().Msgf("Found output wide flag set to: %t", OL.OutputWide)

		jobs, err := GetAPIClient().List(ctx, OL.IDFilter, OL.MaxJobs, OL.ReturnAll, OL.SortBy.String(), OL.SortReverse)
		if err != nil {
			Fatal(fmt.Sprintf("Error listing jobs: %s", err), 1)
		}

		numberInTable := system.Min(OL.MaxJobs, len(jobs))
		log.Debug().Msgf("Number of jobs printing: %d", numberInTable)

		var msgBytes []byte
		if OL.OutputFormat == JSONFormat {
			msgBytes, err = model.JSONMarshalWithMax(jobs)
			if err != nil {
				Fatal(fmt.Sprintf("Error marshaling jobs to JSON: %s", err), 1)
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

			rows := []table.Row{}
			for _, j := range jobs {
				var summaryRow table.Row
				summaryRow, err = summarizeJob(ctx, j)
				if err != nil {
					Fatal(fmt.Sprintf("Error summarizing job: %s", err), 1)
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
	},
}

// Renders job details into a table row
func summarizeJob(ctx context.Context, j *model.Job) (table.Row, error) {
	//nolint:ineffassign,staticcheck // For tracing
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/list.summarizeJob")
	defer span.End()

	jobDesc := []string{
		j.Spec.Engine.String(),
	}
	// Add more details to the job description (e.g. Docker ubuntu echo Hello World)
	if j.Spec.Engine == model.EngineDocker {
		jobDesc = append(jobDesc, j.Spec.Docker.Image, strings.Join(j.Spec.Docker.Entrypoint, " "))
	}

	// compute state summary
	//nolint:gocritic
	stateSummary := job.ComputeStateSummary(j)

	// compute verifiedSummary
	verifiedSummary := job.ComputeVerifiedSummary(j)

	// compute resultSummary
	resultSummary := job.ComputeResultsSummary(j)

	row := table.Row{
		shortenTime(OL.OutputWide, j.CreatedAt),
		shortID(OL.OutputWide, j.ID),
		shortenString(OL.OutputWide, strings.Join(jobDesc, " ")),
		shortenString(OL.OutputWide, stateSummary),
		shortenString(OL.OutputWide, verifiedSummary),
		shortenString(OL.OutputWide, resultSummary),
	}

	return row, nil
}
