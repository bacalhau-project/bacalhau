package internal

import (
	"context"
	"fmt"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/rs/zerolog/log"
)

type JobServer struct {
	RequesterNode *RequesterNode
}

func (server *JobServer) List(args *types.ListArgs, reply *types.ListResponse) error {
	res, err := server.RequesterNode.Scheduler.List()
	if err != nil {
		return err
	}
	*reply = res
	return nil
}

func (server *JobServer) Submit(args *types.SubmitArgs, reply *types.Job) error {
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
		log.Fatal().Msgf("Format of service Job isn't correct. %s", err)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: rpcServer,
	}

	isClosing := false
	go func() {
		err = httpServer.ListenAndServe()
		if err != nil && !isClosing {
			log.Fatal().Msgf("http.ListenAndServe failed: %s", err)
		}
	}()

	go func() {
		log.Debug().Msg("Waiting for json rpc context to finish.")
		<-ctx.Done()
		log.Debug().Msg("Closing json rpc server.")
		isClosing = true
		httpServer.Close()
		log.Debug().Msg("Closed json rpc server.")
	}()
}
