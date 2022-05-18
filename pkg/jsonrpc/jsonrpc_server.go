package jsonrpc

import (
	"fmt"
	"net/http"
	"net/rpc"

	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
)

type JSONRpcServer struct {
	cancelContext *system.CancelContext
	Host          string
	Port          int
	RequesterNode *requestor_node.RequesterNode
}

func NewBacalhauJsonRpcServer(
	cancelContext *system.CancelContext,
	host string,
	port int,
	requesterNode *requestor_node.RequesterNode,
) *JSONRpcServer {
	server := &JSONRpcServer{
		cancelContext: cancelContext,
		Host:          host,
		Port:          port,
		RequesterNode: requesterNode,
	}
	return server
}

// this is not a method of the JSONRpcServer because
// those methods are actual JSONRPC methods and this is just an internal bootstrap
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
		log.Debug().Msg("Json rpc server has started")
	}()

	server.cancelContext.AddShutdownHandler(func() {
		isClosing = true
		httpServer.Close()
		log.Debug().Msg("Json rpc server has stopped")
	})

	return nil
}

func (server *JSONRpcServer) List(args *types.ListArgs, reply *types.ListResponse) error {
	res, err := server.RequesterNode.Transport.List()
	if err != nil {
		return err
	}
	*reply = res
	return nil
}

func (server *JSONRpcServer) Submit(args *types.SubmitArgs, reply *types.Job) error {
	//nolint
	job, err := server.RequesterNode.Transport.SubmitJob(args.Spec, args.Deal)
	if err != nil {
		return err
	}
	*reply = *job
	return nil
}
