package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/traces"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

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

		data, err := getJobResults(cmdArgs[0])
		if err != nil {
			return err
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
		t.AppendHeader(table.Row{"NODE", "IPFS", "RESULTS", "DIFFERENCE"})
		t.SetColumnConfigs([]table.ColumnConfig{})

		clustered := traces.TraceCollection{
			Traces: []traces.Trace{},
		}

		for _, row := range *data {
			resultsFolder, err := system.GetSystemDirectory(row.Folder)
			if err != nil {
				return err
			}

			if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
				fmt.Printf("continue not exist\n")
				continue
			}
			clustered.Traces = append(clustered.Traces, traces.Trace{
				ResultId: row.Cid,
				Filename: resultsFolder + "/metrics.log",
			})
		}

		scores, err := clustered.Scores()
		if err != nil {
			return err
		}

		for _, row := range *data {
			resultsFolder, err := system.GetSystemDirectory(row.Folder)
			if err != nil {
				return err
			}
			folderString := ""
			if _, err := os.Stat(resultsFolder); !os.IsNotExist(err) {
				folderString = fmt.Sprintf("~/.bacalhau/%s", row.Folder)
			}
			t.AppendRows([]table.Row{
				{
					row.Node,
					fmt.Sprintf("https://ipfs.io/ipfs/%s", row.Cid),
					folderString,
					scores[row.Cid]["real"],
				},
			})
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
