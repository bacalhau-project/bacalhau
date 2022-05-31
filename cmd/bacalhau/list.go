package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/executor"
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
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		jobs, err := getAPIClient().List()
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
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"id", "job", "inputs", "outputs", "concurrency", "node", "state", "result"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, AutoMerge: true},
			{Number: 2, AutoMerge: true},
			{Number: 3, AutoMerge: true},
			{Number: 4, AutoMerge: true},
			{Number: 5, AutoMerge: true},
		})

		for _, job := range jobs {
			jobDesc := []string{
				job.Spec.Engine,
			}

			if job.Spec.Engine == string(executor.EXECUTOR_DOCKER) {
				jobDesc = append(jobDesc, job.Spec.Vm.Image)
				jobDesc = append(jobDesc, strings.Join(job.Spec.Vm.Entrypoint, " "))
			}

			if len(job.State) == 0 {
				t.AppendRows([]table.Row{
					{
						shortId(job.Id),
						shortenString(strings.Join(jobDesc, " ")),
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
							len(job.Spec.Inputs),
							len(job.Spec.Outputs),
							job.Deal.Concurrency,
							shortId(node),
							shortenString(jobState.State),
							shortenString(getJobResult(job, jobState)),
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
