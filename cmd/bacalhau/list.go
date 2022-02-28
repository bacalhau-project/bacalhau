package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/types"
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

func ListJobs(
	rpcHost string,
	rpcPort int,
) (*types.ListResponse, error) {
	args := &internal.ListArgs{}
	result := &types.ListResponse{}
	err := JsonRpcMethodWithConnection(rpcHost, rpcPort, "List", args, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		result, err := ListJobs(jsonrpcHost, jsonrpcPort)

		if err != nil {
			return err
		}

		if listOutputFormat == "json" {
			msgBytes, err := json.Marshal(result)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", msgBytes)
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

		for _, job := range result.Jobs {
			for node := range result.JobState[job.Id] {
				t.AppendRows([]table.Row{
					{
						shortId(job.Id),
						getString(strings.Join(job.Commands, "\n")),
						getString(strings.Join(job.Cids, "\n")),
						getString(node),
						result.JobState[job.Id][node],
						result.JobStatus[job.Id][node],
						result.JobResults[job.Id][node],
					},
				})
			}
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
