package jsonrpc

import (
	"context"
	"fmt"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type JSONRpcServer struct {
	Ctx           context.Context
	Host          string
	Port          int
	RequesterNode *requestor_node.RequesterNode
}

func NewBacalhauJsonRpcServer(
	ctx context.Context,
	host string,
	port int,
	requesterNode *requestor_node.RequesterNode,
) *JSONRpcServer {
	server := &JSONRpcServer{
		Ctx:           ctx,
		Host:          host,
		Port:          port,
		RequesterNode: requesterNode,
	}
	return server
}

func StartBacalhauJsonRpcServer(server *JSONRpcServer) error {
	rpcServer := rpc.NewServer()
	err := rpcServer.Register(server)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", server.Host, server.Port),
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
		<-server.Ctx.Done()
		log.Debug().Msg("Closing json rpc server")
		isClosing = true
		httpServer.Close()
		log.Debug().Msg("Closed json rpc server")
	}()

	return nil
}

func (server *JSONRpcServer) List(args *types.ListArgs, reply *types.ListResponse) error {
	res, err := server.RequesterNode.Scheduler.List()
	if err != nil {
		return err
	}
	*reply = res
	return nil
}

func (server *JSONRpcServer) Submit(args *types.SubmitArgs, reply *types.Job) error {
	//nolint
	job, err := server.RequesterNode.Scheduler.SubmitJob(args.Spec, args.Deal)
	if err != nil {
		return err
	}
	*reply = *job
	return nil
}
