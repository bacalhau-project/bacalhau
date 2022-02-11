package bacalhau

import (
	"fmt"
	"log"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var jobCids []string
var jobCommands []string

func init() {
	submitCmd.PersistentFlags().StringSliceVar(
		&jobCids, "cids", []string{},
		`The cids of the data used by the job (comma separated, or specify multiple times)`,
	)
	submitCmd.PersistentFlags().StringSliceVar(
		&jobCommands, "commands", []string{},
		`The commands for the job (comma separated, or specify multiple times)`,
	)
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		jobUuid, err := uuid.NewRandom()
		if err != nil {
			log.Fatalf("Error in creating job id. %s", err)
		}

		if len(jobCommands) <= 0 {
			log.Fatalf("Empty command list")
		}

		job := &types.Job{
			Id:       jobUuid.String(),
			Cpu:      1,
			Memory:   2,
			Disk:     10,
			Cids:     jobCids,
			Commands: jobCommands,
		}

		args := &internal.SubmitArgs{
			Job: job,
		}
		result := &types.Job{}

		err = JsonRpcMethod("Submit", args, result)
		if err != nil {
			return err
		}

		//we got our result in result
		// fmt.Printf("submit job: %+v\nreply job: %+v\n\n", args.Job, result)
		// fmt.Printf("to view all files by all nodes\n")
		// fmt.Printf("------------------------------\n\n")
		// fmt.Printf("tree ./outputs/%s\n\n", job.Id)
		// fmt.Printf("to open all metrics pngs\n")
		// fmt.Printf("------------------------\n\n")
		// fmt.Printf("find ./outputs/%s -type f -name 'metrics.png' 2> /dev/null | while read -r FILE ; do xdg-open \"$FILE\" ; done\n\n", job.Id)
		fmt.Printf("job id: %s\n", job.Id)
		return nil
	},
}
