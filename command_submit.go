package main

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/spf13/cobra"
)

//Result of RPC call is of this type
type Result int

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
			Id: cmdArgs[0],
		}
		//make arguments object
		args := &SubmitArgs{
			Job: job,
		}
		//this will store returned result
		var result Result
		//call remote procedure with args
		err = client.Call("JobServer.Submit", args, &result)
		if err != nil {
			log.Fatalf("error in JobServer", err)
		}
		//we got our result in result
		log.Printf("submit job: %+v\nreply job: %+v\n", args.Job, result)
		return nil
	},
}
