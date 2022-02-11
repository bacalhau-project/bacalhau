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

type ResultsList struct {
	Node string
	Cid  string
}

func init() {
	resultsListCmd.PersistentFlags().StringVar(
		&listOutputFormat, "output", "text",
		`The output format for the list of jobs (json or text)`,
	)
}

var resultsListCmd = &cobra.Command{
	Use:   "list [job_id]",
	Short: "List results for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		jobId := cmdArgs[0]

		args := &internal.ListArgs{}
		result := &types.ListResponse{}
		err := JsonRpcMethod("List", args, result)
		if err != nil {
			return err
		}

		var foundJob *types.Job

		for _, job := range result.Jobs {
			if strings.HasPrefix(job.Id, jobId) {
				foundJob = &job
			}
		}

		if foundJob == nil {
			return fmt.Errorf("Could not find job: %s", jobId)
		}

		data := []ResultsList{}
		for node := range result.JobState[foundJob.Id] {
			data = append(data, ResultsList{
				Node: node,
				Cid:  fmt.Sprintf("https://ipfs.io/ipfs/%s", result.JobResults[foundJob.Id][node]),
			})
		}

		if listOutputFormat == "json" {
			msgBytes, err := json.Marshal(data)
			if err != nil {
				return err
			}
			fmt.Printf("%s\n", msgBytes)
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"NODE", "DATA"})
		t.SetColumnConfigs([]table.ColumnConfig{})

		for _, row := range data {
			t.AppendRows([]table.Row{
				{
					row.Node,
					row.Cid,
				},
			})
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
