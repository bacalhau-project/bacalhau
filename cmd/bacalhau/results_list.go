package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/rs/zerolog/log"
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
	RunE: func(cmd *cobra.Command, cmdArgs []string) error { // nolint

		_, err := getJobData(cmdArgs[0])
		if err != nil {
			return err
		}

		data, err := getJobResults(cmdArgs[0])
		if err != nil {
			return err
		}

		if listOutputFormat == "json" {
			msgBytes, err := json.Marshal(data)
			if err != nil {
				return err
			}
			log.Debug().Msg(fmt.Sprintf("Result list msgBytes: %s\n", msgBytes))
			return nil
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"NODE", "IPFS", "RESULTS"})
		t.SetColumnConfigs([]table.ColumnConfig{})

		for _, row := range *data {
			resultsFolder, err := system.GetSystemDirectory(row.Folder)
			if err != nil {
				return err
			}

			if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
				log.Warn().Msg("Results folder does not exist, continuing.")
				continue
			}
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
				},
			})
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
