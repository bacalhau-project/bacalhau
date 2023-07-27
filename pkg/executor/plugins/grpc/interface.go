package grpc

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/plugins/grpc/proto"
)

type ExecutorGRPCPlugin struct {
	plugin.Plugin
	Impl executor.Executor
}

func (p *ExecutorGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	proto.RegisterExecutorServer(s, &GRPCServer{Impl: p.Impl})
	return nil
}

func (p *ExecutorGRPCPlugin) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &GRPCClient{client: proto.NewExecutorClient(c)}, nil
}
