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
	RequesterNode *RequesterNode
}

type ListArgs struct {
}

type SubmitArgs struct {
	Spec *types.JobSpec
	Deal *types.JobDeal
}

func (server *JobServer) List(args *ListArgs, reply *types.ListResponse) error {
	res, err := server.RequesterNode.Scheduler.List()
	if err != nil {
		return err
	}
	*reply = res
	return nil
}

func (server *JobServer) Submit(args *SubmitArgs, reply *types.Job) error {
	//nolint
	job, err := server.RequesterNode.Scheduler.SubmitJob(args.Spec, args.Deal)
	if err != nil {
		return err
	}
	*reply = *job
	return nil
}

func RunBacalhauJsonRpcServer(
	ctx context.Context,
	host string,
	port int,
	requesterNode *RequesterNode,
) {
	job := &JobServer{
		RequesterNode: requesterNode,
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

	go func() {
		fmt.Printf("waiting for json rpc context done\n")
		<-ctx.Done()
		fmt.Printf("closing json rpc server\n")
		isClosing = true
		httpServer.Close()
		fmt.Printf("closed json rpc server\n")
	}()
}
