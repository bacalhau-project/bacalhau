package main

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/spf13/cobra"
)

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
			Id:     cmdArgs[0],
			Cpu:    1,
			Memory: 1,
			Disk:   10,
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
