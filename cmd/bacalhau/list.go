package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
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
			msgBytes, err := json.Marshal(result)
			if err != nil {
				return err
			}
			log.Debug().Msg(fmt.Sprintf("List msg bytes: %s\n", msgBytes))
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"JOB", "COMMAND", "DATA", "NODE", "STATE", "STATUS", "OUTPUT"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, AutoMerge: true},
			{Number: 2, AutoMerge: true},
			{Number: 3, AutoMerge: true},
		})

		for _, jobData := range result.Jobs {
			inputCids := []string{}
			for _, input := range jobData.Spec.Inputs {
				inputCids = append(inputCids, input.Cid)
			}

			for node, jobState := range jobData.State {
				t.AppendRows([]table.Row{
					{
						shortId(jobData.Id),
						jobData.Spec.Vm.Entrypoint,
						shortenString(strings.Join(inputCids, "\n")),
						shortenString(node),
						jobState.State,
						jobState.Status,
						jobState.ResultsId,
					},
				})
			}
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
