package internal

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal/types"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type SubmitArgs struct {
	Job *types.Job
}

func (server *JobServer) Submit(args *SubmitArgs, reply *types.Job) error {
	server.ComputeNode.Publish(args.Job)
	*reply = *args.Job
	return nil
}

func RunBacalhauRpcServer(host string, port int, computeNode *ComputeNode) {
	job := &JobServer{
		ComputeNode: computeNode,
	}
	err := rpc.Register(job)
	if err != nil {
		log.Fatalf("Format of service Job isn't correct. %s", err)
	}
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if e != nil {
		log.Fatalf("Couldn't start listening on port %d. Error %s", port, e)
	}
	log.Println("Serving RPC handler")
	err = http.Serve(l, nil)
	if err != nil {
		log.Fatalf("Error serving: %s", err)
	}
}
