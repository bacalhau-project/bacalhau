package bacalhau

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/traces"
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

		job, err := getJobData(cmdArgs[0])
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
		t.AppendHeader(table.Row{"NODE", "IPFS", "RESULTS", "DIFFERENCE", "CORRECT"})
		t.SetColumnConfigs([]table.ColumnConfig{})

		log.Debug().Msg(fmt.Sprintf("Job deal tolerance: %f\n", job.Deal.Tolerance))

		// TODO: load the job so we can get at Deal.Tolerance
		clustered := traces.TraceCollection{
			Traces:    []traces.Trace{},
			Tolerance: job.Deal.Tolerance,
		}

		for _, row := range *data {
			resultsFolder, err := system.GetSystemDirectory(row.Folder)
			if err != nil {
				return err
			}

			if _, err := os.Stat(resultsFolder); os.IsNotExist(err) {
				log.Warn().Msg("Results folder does not exist, continuing.")
				continue
			}
			clustered.Traces = append(clustered.Traces, traces.Trace{
				ResultId: row.Cid,
				Filename: resultsFolder + "/metrics.log",
			})
		}

		correctGroup, incorrectGroup, _ := clustered.Cluster()

		log.Info().Msg(fmt.Sprintf(`
Returned results:
	Correct: %+v
	Incorrect: %+v`, correctGroup, incorrectGroup))

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

			correctStatus := "❌"

			for _, correctId := range correctGroup {
				if correctId == row.Cid {
					correctStatus = "✅"
				}
			}

			t.AppendRows([]table.Row{
				{
					row.Node,
					fmt.Sprintf("https://ipfs.io/ipfs/%s", row.Cid),
					folderString,
					scores[row.Cid]["real"],
					correctStatus,
				},
			})
		}
		t.SetStyle(table.StyleColoredGreenWhiteOnBlack)
		t.Render()

		return nil
	},
}
