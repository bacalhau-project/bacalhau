package main

import (
	"fmt"
	"log"
	"net/rpc"
	"os"

	"github.com/spf13/cobra"
)

//Result of RPC call is of this type
type Result int

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit a job to the network",
	RunE: func(cmd *cobra.Command, _ []string) error {

		//make connection to rpc server
		client, err := rpc.DialHTTP("tcp", fmt.Sprintf(":%d", jsonrpcPort))
		if err != nil {
			log.Fatalf("Error in dialing. %s", err)
		}
		//make arguments object
		args := &Args{
			A: 2,
			B: 3,
		}
		//this will store returned result
		var result Result
		//call remote procedure with args
		err = client.Call("Arith.Multiply", args, &result)
		if err != nil {
			log.Fatalf("error in Arith", err)
		}
		//we got our result in result
		log.Printf("%d*%d=%d\n", args.A, args.B, result)

		fmt.Printf("hello world\n")
		os.Exit(0)
		return nil
	},
}
