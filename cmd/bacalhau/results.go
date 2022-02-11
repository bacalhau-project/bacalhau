package bacalhau

import (
	"github.com/spf13/cobra"
)

func init() {
	resultsCmd.AddCommand(resultsListCmd)
	resultsCmd.AddCommand(resultsFetchCmd)
}

var resultsCmd = &cobra.Command{
	Use:   "results",
	Short: "Get results for jobs",
}
