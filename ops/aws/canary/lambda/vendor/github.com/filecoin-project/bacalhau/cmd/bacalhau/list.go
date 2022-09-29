package bacalhau

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

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
		`reverse order of table - for time sorting, this will be newest first.`)

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
	RunE: func(cmd *cobra.Command, args []string) error {
		cm := system.NewCleanupManager()
		defer cm.Cleanup()
		ctx := cmd.Context()

		t := system.GetTracer()
		ctx, rootSpan := system.NewRootSpan(ctx, t, "cmd/bacalhau/list")
		defer rootSpan.End()
		cm.RegisterCallback(system.CleanupTraceProvider)

		jobs, err := getAPIClient().List(ctx)
		if err != nil {
			return err
		}

		tw := table.NewWriter()
		tw.SetOutputMirror(cmd.OutOrStderr())
		if !OL.HideHeader {
			tw.AppendHeader(table.Row{"created", "id", "job", "state", "verified", "published"})
		}

		columnConfig := []table.ColumnConfig{}

		tw.SetColumnConfigs(columnConfig)

		jobArray := []model.Job{}
		for _, j := range jobs {
			if OL.IDFilter != "" {
				if j.ID == OL.IDFilter || shortID(false, j.ID) == OL.IDFilter {
					jobArray = append(jobArray, j)
				}
			} else {
				jobArray = append(jobArray, j)
			}
		}

		log.Debug().Msgf("Found table sort flag: %s", OL.SortBy)
		log.Debug().Msgf("Table filter flag set to: %s", OL.IDFilter)
		log.Debug().Msgf("Table reverse flag set to: %t", OL.SortReverse)

		sort.Slice(jobArray, func(i, j int) bool {
			switch OL.SortBy {
			case ColumnID:
				return shortID(OL.OutputWide, jobArray[i].ID) < shortID(OL.OutputWide, jobArray[j].ID)
			case ColumnCreatedAt:
				return jobArray[i].CreatedAt.Format(time.RFC3339) < jobArray[j].CreatedAt.Format(time.RFC3339)
			default:
				return false
			}
		})

		if OL.SortReverse {
			jobIDs := []string{}
			for _, j := range jobArray {
				jobIDs = append(jobIDs, j.ID)
			}
			jobIDs = ReverseList(jobIDs)
			jobArray = []model.Job{}
			for _, id := range jobIDs {
				jobArray = append(jobArray, jobs[id])
			}
		}

		numberInTable := Min(OL.MaxJobs, len(jobArray))

		log.Debug().Msgf("Number of jobs printing: %d", numberInTable)

		rows, err := resolvingJobDetails(ctx, jobArray, numberInTable)
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

		if OL.OutputFormat == JSONFormat {
			msgBytes, err := json.MarshalIndent(jobs, "", "    ")
			if err != nil {
				return err
			}

			cmd.Printf("%s\n", msgBytes)
			return nil
		} else {
			tw.Render()
		}

		return nil
	},
}

func resolvingJobDetails(ctx context.Context,
	jobArray []model.Job,
	lengthOfTable int) ([]table.Row, error) {
	ctx, span := system.GetTracer().Start(ctx, "cmd/bacalhau/list.resolvingJobDetails")
	defer span.End()

	rows := []table.Row{}

	// using indexing to avoid copying bytes
	for i := range jobArray[0:lengthOfTable] {
		jobDesc := []string{
			jobArray[i].Spec.Engine.String(),
		}

		if jobArray[i].Spec.Engine == model.EngineDocker {
			jobDesc = append(jobDesc, jobArray[i].Spec.Docker.Image, strings.Join(jobArray[i].Spec.Docker.Entrypoint, " "))
		}

		resolver := getAPIClient().GetJobStateResolver()

		stateSummary, err := resolver.StateSummary(ctx, jobArray[i].ID)
		if err != nil {
			return nil, err
		}

		resultSummary, err := resolver.ResultSummary(ctx, jobArray[i].ID)
		if err != nil {
			return nil, err
		}

		verifiedSummary, err := resolver.VerifiedSummary(ctx, jobArray[i].ID)
		if err != nil {
			return nil, err
		}

		rows = append(rows, table.Row{
			shortenTime(OL.OutputWide, jobArray[i].CreatedAt),
			shortID(OL.OutputWide, jobArray[i].ID),
			shortenString(OL.OutputWide, strings.Join(jobDesc, " ")),
			shortenString(OL.OutputWide, stateSummary),
			shortenString(OL.OutputWide, verifiedSummary),
			shortenString(OL.OutputWide, resultSummary),
		})
	}
	return rows, nil
}
