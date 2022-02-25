package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

type JobServer struct {
	ComputeNode *ComputeNode
}

type ListArgs struct {
}

type SubmitArgs struct {
	Job *types.Job
}

func (server *JobServer) List(r *http.Request, args *ListArgs, reply *types.ListResponse) error {
	*reply = types.ListResponse{
		Jobs:       server.ComputeNode.Jobs,
		JobState:   server.ComputeNode.JobState,
		JobStatus:  server.ComputeNode.JobStatus,
		JobResults: server.ComputeNode.JobResults,
	}
	return nil
}

func (server *JobServer) Submit(r *http.Request, args *SubmitArgs, reply *types.Job) error {
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

	server := rpc.NewServer()
	server.RegisterCodec(json.NewCodec(), "application/json")
	server.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	err := server.RegisterService(job, "")
	if err != nil {
		log.Fatalf("server.RegisterService failed: %s", err)
	}
	router := mux.NewRouter()
	router.Handle("/", server)
	httpServer := &http.Server{Addr: fmt.Sprintf("%s:%d", host, port), Handler: router}

	isClosing := false
	go func() {
		err = httpServer.ListenAndServe()
		if err != nil && !isClosing {
			log.Fatalf("http.ListenAndServe failed: %s", err)
		}
	}()

	<-ctx.Done()

	isClosing = true

	httpServer.Close()
}
