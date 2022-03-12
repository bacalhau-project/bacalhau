package bacalhau

import (
	"github.com/spf13/cobra"
)

func init() {

}

var resultsFetchCmd = &cobra.Command{
	Use:   "fetch [job_id]",
	Short: "Fetch results for a job",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {
		return fetchJobResults(cmdArgs[0])
	},
}
