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
	"github.com/filecoin-project/bacalhau/pkg/job"
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
	listCmd.PersistentFlags().BoolVar(&tableSortReverse, "reverse", true,
		`reverse order of table - for time sorting, this will be newest first.`)

	listCmd.PersistentFlags().Var(&tableSortBy, "sort-by",
		`sort by field, defaults to creation time, with newest first [Allowed "id", "created_at"].`)
	listCmd.PersistentFlags().Lookup("sort-by").DefValue = string(ColumnCreatedAt)
	if tableSortBy == "" {
		tableSortBy = ColumnCreatedAt
	}

	listCmd.PersistentFlags().BoolVar(
		&tableOutputWide, "wide", false,
		`Print full values in the table results`,
	)
	// listCmd.PersistentFlags().BoolVar(
	// 	&tableMergeValues, "merge-identical", false,
	// 	`Merge identical values`,
	// )
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

		t := table.NewWriter()
		t.SetOutputMirror(cmd.OutOrStderr())
		if !tableHideHeader {
			t.AppendHeader(table.Row{"creation_time", "id", "job", "state", "result"})
		}

		columnConfig := []table.ColumnConfig{}

		t.SetColumnConfigs(columnConfig)

		jobArray := []*executor.Job{}
		for _, thisJob := range jobs {
			if tableIdFilter != "" {
				if thisJob.Id == tableIdFilter || shortId(thisJob.Id) == tableIdFilter {
					jobArray = append(jobArray, thisJob)
				}
			} else {
				jobArray = append(jobArray, thisJob)
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

		for _, jobInRow := range jobArray[0:numberInTable] {
			jobDesc := []string{
				jobInRow.Spec.Engine.String(),
			}

			if jobInRow.Spec.Engine == executor.EngineDocker {
				jobDesc = append(jobDesc, jobInRow.Spec.Vm.Image)
				jobDesc = append(jobDesc, strings.Join(jobInRow.Spec.Vm.Entrypoint, " "))
			}

			if len(jobInRow.State) == 0 {
				t.AppendRows([]table.Row{
					{
						jobInRow.CreatedAt.Format("06-01-02-15:04:05"),
						shortId(jobInRow.Id),
						shortenString(strings.Join(jobDesc, " ")),
						"waiting",
						"",
					},
				})
			} else {
				_, currentJobState := job.GetCurrentJobState(jobInRow)
				t.AppendRows([]table.Row{
					{
						shortenTime(jobInRow.CreatedAt),
						shortId(jobInRow.Id),
						shortenString(strings.Join(jobDesc, " ")),
						shortenString(currentJobState.State.String()),
						shortenString(getJobResult(jobInRow, currentJobState)),
					},
				})
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

		if listOutputFormat == "json" {
			msgBytes, err := json.MarshalIndent(jobs, "", "    ")
			if err != nil {
				return err
			}

			fmt.Printf("%s\n", msgBytes)
			return nil
		} else {
			t.Render()
		}

		return nil
	},
}
