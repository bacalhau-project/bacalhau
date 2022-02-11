package bacalhau

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {

}

var resultsFetchCmd = &cobra.Command{
	Use:   "fetch <job_id>",
	Short: "Fetch results for a job",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		fmt.Printf("HELLO WORLD\n")
		return nil
	},
}
