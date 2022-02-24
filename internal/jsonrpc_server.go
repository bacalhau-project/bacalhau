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

type ListArgs struct {
}

type ListResponse struct {
	Jobs       []types.Job
	JobState   map[string]map[string]string
	JobStatus  map[string]map[string]string
	JobResults map[string]map[string]string
}

type SubmitArgs struct {
	Job *types.Job
}

func (server *JobServer) List(args *ListArgs, reply *ListResponse) error {
	*reply = ListResponse{
		Jobs:       server.ComputeNode.Jobs,
		JobState:   server.ComputeNode.JobState,
		JobStatus:  server.ComputeNode.JobStatus,
		JobResults: server.ComputeNode.JobResults,
	}
	return nil
}

func (server *JobServer) Submit(args *SubmitArgs, reply *types.Job) error {
	//nolint
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
