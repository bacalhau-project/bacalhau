package bacalhau

import (
	"encoding/json"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"strings"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

var listOutputFormat string
var tableOutputWide bool

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

func getString(st string) string {
	if tableOutputWide {
		return st
	}

	if len(st) < 20 {
		return st
	}

	return st[:20] + "..."
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List jobs on the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		// make connection to rpc server
		client, err := rpc.DialHTTP("tcp", fmt.Sprintf(":%d", jsonrpcPort))
		if err != nil {
			log.Fatalf("Error in dialing. %s", err)
		}
		args := &internal.ListArgs{}
		result := &internal.ListResponse{}
		err = client.Call("JobServer.List", args, result)
		if err != nil {
			log.Fatalf("error in JobServer: %s", err)
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
						getString(job.Id),
						getString(strings.Join(job.Commands, "\n")),
						getString(strings.Join(job.Cids, "\n")),
						getString(node),
						result.JobState[job.Id][node],
						result.JobStatus[job.Id][node],
						getString(result.JobResults[job.Id][node]),
					},
				})
			}
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
