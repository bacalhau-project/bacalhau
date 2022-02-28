package bacalhau

import (
	"fmt"
	"os"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/spf13/cobra"
)

func init() {

}

var resultsFetchCmd = &cobra.Command{
	Use:   "fetch [job_id]",
	Short: "Fetch results for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		data, err := getJobResults(cmdArgs[0])
		if err != nil {
			return err
		}

		for _, row := range *data {
			resultsFolder, err := system.GetSystemDirectory(row.Folder)
			if err != nil {
				return err
			}
			if _, err := os.Stat(resultsFolder); !os.IsNotExist(err) {
				continue
			}
			fmt.Printf("Fetching results for job %s ---> %s\n", row.Cid, row.Folder)
			resultsFolder, err = system.EnsureSystemDirectory(row.Folder)
			if err != nil {
				return err
			}
			err = system.RunCommand("ipfs", []string{
				"get",
				row.Cid,
				"--output",
				resultsFolder,
			})
			if err != nil {
				return err
			}
		}

		return nil
	},
}
