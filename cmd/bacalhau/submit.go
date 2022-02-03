package bacalhau

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

var jobId string

func init() {
	submitCmd.PersistentFlags().StringVar(
		&jobId, "id", "",
		`The id of the job to submit`,
	)
}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, cmdArgs []string) error {

		//make connection to rpc server
		client, err := rpc.DialHTTP("tcp", fmt.Sprintf(":%d", jsonrpcPort))
		if err != nil {
			log.Fatalf("Error in dialing. %s", err)
		}

		if jobId == "" {
			jobUuid, err := uuid.NewRandom()
			if err != nil {
				log.Fatalf("Error in creating job id. %s", err)
			}
			jobId = jobUuid.String()
		}

		job := &types.Job{
			Id:     jobId,
			Cpu:    1,
			Memory: 2,
			Disk:   10,
			Commands: []string{
				// "unzip 5m-Sales-Records.zip",
				// "for X in {1..10}; do bash -c \"sed 's/Office Supplies/Booze/' '5m Sales Records.csv' -i\"; sleep 2; done",
				"echo HELLO THIS IS THE EXECUTION STEP",
				"for X in {1..10}; do echo iteration $X; for Y in {0..100000}; do false; done; sleep 2; done",
				"echo DONE",
			},
		}
		args := &internal.SubmitArgs{
			Job: job,
		}
		result := types.Job{}
		err = client.Call("JobServer.Submit", args, &result)
		if err != nil {
			log.Fatalf("error in JobServer: %s", err)
		}
		//we got our result in result
		log.Printf("submit job: %+v\nreply job: %+v\n", args.Job, result)
		log.Printf("to see the results once they have been created\n\n---------------------\n\nls -la outputs/%s\n\n---------------------\n", job.Id)
		return nil
	},
}
