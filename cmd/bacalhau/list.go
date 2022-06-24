package bacalhau

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.PersistentFlags().BoolVar(&tableHideHeader, "hide-header", false,
		`do not print the column headers.`)
	listCmd.PersistentFlags().StringVar(&tableIdFilter, "id-filter", "", `filter by Job List to IDs matching substring.`)
	listCmd.PersistentFlags().BoolVar(&tableNoStyle, "no-style", false, `remove all styling from table output.`)
	listCmd.PersistentFlags().IntVarP(
		&tableMaxJobs, "number", "n", 10,
		`print the first NUM jobs instead of the first 10.`,
	)
	listCmd.PersistentFlags().StringVar(
		&listOutputFormat, "output", "text",
		`The output format for the list of jobs (json or text)`,
	)
	listCmd.PersistentFlags().BoolVar(&tableSortReverse, "reverse", false,
		`reverse order of table - for time sorting, this will be newest first.`)
	listCmd.PersistentFlags().Var(&tableSortBy, "sort-by",
		`sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].`)
	listCmd.PersistentFlags().BoolVar(
		&tableOutputWide, "wide", false,
		`Print full values in the table results`,
	)
	listCmd.PersistentFlags().BoolVar(
		&tableMergeValues, "merge-identical", false,
		`Merge identical values`,
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
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		jobs, err := getAPIClient().List(context.Background())
		if err != nil {
			return err
		}
		if listOutputFormat == "json" {
			msgBytes, err := json.MarshalIndent(jobs, "", "    ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", msgBytes)
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStderr())
		if !tableHideHeader {
			t.AppendHeader(table.Row{"id", "job", "creation_time", "inputs", "outputs", "concurrency", "node", "state", "result"})
		}

		columnConfig := []table.ColumnConfig{}

		if tableMergeValues {

			// don't merge node, state and result
			// because they should differentiate even for the same job
			columnConfig = []table.ColumnConfig{
				{Number: 1, AutoMerge: true},
				{Number: 2, AutoMerge: true},
				{Number: 3, AutoMerge: true},
				{Number: 4, AutoMerge: true},
				{Number: 5, AutoMerge: true},
				{Number: 6, AutoMerge: true},
			}
		}

		t.SetColumnConfigs(columnConfig)

		jobArray := []*executor.Job{}
		for _, job := range jobs {
			if tableIdFilter != "" {
				if job.Id == tableIdFilter || shortId(job.Id) == tableIdFilter {
					jobArray = append(jobArray, job)
				}
			} else {
				jobArray = append(jobArray, job)
			}
		}

		log.Debug().Msgf("Found table sort flag: %s", tableSortBy)
		log.Debug().Msgf("Table filter flag set to: %s", tableIdFilter)
		log.Debug().Msgf("Table reverse flag set to: %t", tableSortReverse)

		sort.Slice(jobArray, func(i, j int) bool {
			switch tableSortBy {
			case ColumnID:
				return shortId(jobArray[i].Id) < shortId(jobArray[j].Id)
			case ColumnCreatedAt:
				return jobArray[i].CreatedAt.Format(time.RFC3339) < jobArray[j].CreatedAt.Format(time.RFC3339)
			default:
				return false
			}
		})

		if tableSortReverse {
			jobIds := []string{}
			for _, job := range jobArray {
				jobIds = append(jobIds, job.Id)
			}
			jobIds = ReverseList(jobIds)
			jobArray = []*executor.Job{}
			for _, id := range jobIds {
				jobArray = append(jobArray, jobs[id])
			}
		}

		numberInTable := Min(tableMaxJobs, len(jobArray))
		log.Debug().Msgf("Number of jobs printing: %d", numberInTable)

		for _, job := range jobArray[0:numberInTable] {
			jobDesc := []string{
				job.Spec.Engine.String(),
			}

			if job.Spec.Engine == executor.EngineDocker {
				jobDesc = append(jobDesc, job.Spec.Docker.Image)
				jobDesc = append(jobDesc, strings.Join(job.Spec.Docker.Entrypoint, " "))
			}

			if len(job.State) == 0 {
				t.AppendRows([]table.Row{
					{
						shortId(job.Id),
						shortenString(strings.Join(jobDesc, " ")),
						job.CreatedAt.Format("06-01-02-15:04:05"),
						len(job.Spec.Inputs),
						len(job.Spec.Outputs),
						job.Deal.Concurrency,
						"",
						"waiting",
						"",
					},
				})
			} else {
				for node, jobState := range job.State {
					t.AppendRows([]table.Row{
						{
							shortId(job.Id),
							shortenString(strings.Join(jobDesc, " ")),
							job.CreatedAt.Format("06-01-02-15:04:05"),
							len(job.Spec.Inputs),
							len(job.Spec.Outputs),
							job.Deal.Concurrency,
							shortId(node),
							shortenString(jobState.State.String()),
							shortenString(getJobResult(job, jobState)),
						},
					})
				}
			}

		}
		if tableNoStyle {
			t.SetStyle(table.Style{
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
			t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		}
		t.Render()

		return nil
	},
}
