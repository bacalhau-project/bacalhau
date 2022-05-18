package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func init() {
	listCmd.PersistentFlags().StringVar(
		&listOutputFormat, "output", "text",
		`The output format for the list of jobs (json or text)`,
	)
	listCmd.PersistentFlags().BoolVar(
		&tableOutputWide, "wide", false,
		`Print full values in the table results`,
	)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint

		result, err := job.ListJobs(jsonrpcHost, jsonrpcPort)

		if err != nil {
			return err
		}

		if listOutputFormat == "json" {
			msgBytes, err := json.MarshalIndent(result, "", "    ")
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", msgBytes)
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"id", "job", "inputs", "concurrency", "node", "state", "status", "result"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, AutoMerge: true},
			{Number: 2, AutoMerge: true},
			{Number: 3, AutoMerge: true},
			{Number: 4, AutoMerge: true},
		})

		for _, jobData := range result.Jobs {
			inputCids := []string{}
			for _, input := range jobData.Spec.Inputs {
				inputCids = append(inputCids, shortenString(input.Cid))
			}

			jobDesc := []string{
				jobData.Spec.Engine,
			}

			if jobData.Spec.Engine == executor.EXECUTOR_DOCKER {
				jobDesc = append(jobDesc, jobData.Spec.Vm.Image)
				jobDesc = append(jobDesc, strings.Join(jobData.Spec.Vm.Entrypoint, ""))
			}

			if len(jobData.State) == 0 {
				t.AppendRows([]table.Row{
					{
						shortId(jobData.Id),
						strings.Join(jobDesc, "\n"),
						strings.Join(inputCids, "\n"),
						jobData.Deal.Concurrency,
						"",
						"waiting",
						"",
						"",
					},
				})
			} else {
				for node, jobState := range jobData.State {
					t.AppendRows([]table.Row{
						{
							shortId(jobData.Id),
							strings.Join(jobDesc, "\n"),
							strings.Join(inputCids, "\n"),
							jobData.Deal.Concurrency,
							shortenString(node),
							shortenString(jobState.State),
							shortenString(jobState.Status),
							shortenString(jobState.ResultsId),
						},
					})
				}
			}

		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
