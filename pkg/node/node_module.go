package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

func NewNodeService(cfg NodeConfig) fx.Option {
	return fx.Module("node",
		// we skip this as its already been provided.
		/*
			fx.Provide(func() NodeConfig {
				return cfg
			}),
		*/
		fx.Invoke(setupNode),
	)
}

type NodeDependencies struct {
	fx.In

	ApiServer *publicapi.Server
	Store     routing.NodeInfoStore
	Host      host.Host

	Compute   *Compute   `optional:"true"`
	Requester *Requester `optional:"true"`
}

type NodeService struct {
	fx.Out

	Node *Node
}

func setupNode(lc fx.Lifecycle, cfg NodeConfig, deps NodeDependencies) (NodeService, error) {
	node := &Node{
		APIServer:      deps.ApiServer,
		ComputeNode:    deps.Compute,
		RequesterNode:  deps.Requester,
		NodeInfoStore:  deps.Store,
		CleanupManager: cfg.CleanupManager,
		IPFSClient:     cfg.IPFSClient,
		Host:           deps.Host,
	}
	if node.IsComputeNode() && node.IsRequesterNode() {
		node.ComputeNode.RegisterLocalComputeCallback(node.RequesterNode.localCallback)
		node.RequesterNode.RegisterLocalComputeEndpoint(node.ComputeNode.LocalEndpoint)
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting node")
			return node.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("stopping node")
			return nil
		},
	})
	return NodeService{
		Node: node,
	}, nil

}
