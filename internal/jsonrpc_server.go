package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal/types"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type ListArgs struct {
}

type SubmitArgs struct {
	Job *types.Job
}

func (server *JobServer) List(args *ListArgs, reply *types.ListResponse) error {
	*reply = types.ListResponse{
		Jobs:       server.ComputeNode.Jobs,
		JobState:   server.ComputeNode.JobState,
		JobStatus:  server.ComputeNode.JobStatus,
		JobResults: server.ComputeNode.JobResults,
	}
	return nil
}

func (server *JobServer) Submit(args *SubmitArgs, reply *types.Job) error {
	server.ComputeNode.Publish(args.Job)
	*reply = *args.Job
	return nil
}

func RunBacalhauJsonRpcServer(
	ctx context.Context,
	host string,
	port int,
	computeNode *ComputeNode,
) {
	job := &JobServer{
		ComputeNode: computeNode,
	}

	rpcServer := rpc.NewServer()
	err := rpcServer.Register(job)
	if err != nil {
		log.Fatalf("Format of service Job isn't correct. %s", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: rpcServer,
	}

	isClosing := false
	go func() {
		err = httpServer.ListenAndServe()
		if err != nil && !isClosing {
			log.Fatalf("http.ListenAndServe failed: %s", err)
		}
	}()

	fmt.Printf("waiting for json rpc context done\n")

	<-ctx.Done()

	fmt.Printf("closing json rpc server\n")

	isClosing = true

	httpServer.Close()

	fmt.Printf("closed json rpc server\n")
}
