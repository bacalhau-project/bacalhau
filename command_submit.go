package main

import (
	"fmt"
	"log"
	"net/rpc"

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
		job := &Job{
			Id:     jobId,
			Cpu:    1,
			Memory: 2,
			Disk:   10,
			BuildCommands: []string{
				// "apt update && apt-get install -y unzip",
				// "wget https://eforexcel.com/wp/wp-content/uploads/2020/09/5m-Sales-Records.zip",
				"echo HELLO THIS IS THE BUILD STEP",
			},
			Commands: []string{
				// "unzip 5m-Sales-Records.zip",
				// "for X in {1..10}; do bash -c \"sed 's/Office Supplies/Booze/' '5m Sales Records.csv' -i\"; sleep 2; done",
				"echo HELLO THIS IS THE EXECUTION STEP",
				"for X in {1..10}; do echo iteration $X; for Y in {0..100000}; do false; done; sleep 2; done",
				"echo DONE",
			},
		}
		args := &SubmitArgs{
			Job: job,
		}
		result := Job{}
		err = client.Call("JobServer.Submit", args, &result)
		if err != nil {
			log.Fatalf("error in JobServer", err)
		}
		//we got our result in result
		log.Printf("submit job: %+v\nreply job: %+v\n", args.Job, result)
		return nil
	},
}
