package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type SubmitArgs struct {
	Job *Job
}

func (server *JobServer) Submit(args *SubmitArgs, reply *Job) error {
	server.ComputeNode.Publish(args.Job)
	*reply = *args.Job
	return nil
}

// Represents Arith service for RPC
type Arith int

// Arith service has procedure Multiply which takes numbers A, B as arguments and returns error or stores product in reply
func (t *Arith) Multiply(args *Args, reply *int) error {
	*reply = args.A * args.B
	return nil
}

type Args struct {
	A, B int
}

func runBacalhauRpcServer(port int, computeNode *ComputeNode) {
	job := &JobServer{
		ComputeNode: computeNode,
	}
	err := rpc.Register(job)
	if err != nil {
		log.Fatalf("Format of service Job isn't correct. %s", err)
	}
	// arith := new(Arith)
	// err = rpc.Register(arith)
	// if err != nil {
	// 	log.Fatalf("Format of service Arith isn't correct. %s", err)
	// }
	rpc.HandleHTTP()
	// start listening for messages on port 1234
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if e != nil {
		log.Fatalf("Couldn't start listening on port %d. Error %s", port, e)
	}
	log.Println("Serving RPC handler")
	err = http.Serve(l, nil)
	if err != nil {
		log.Fatalf("Error serving: %s", err)
	}
}
