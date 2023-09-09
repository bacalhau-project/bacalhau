package node

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewComputeService(cfg ComputeConfig) fx.Option {
	return fx.Module("compute",
		fx.Provide(func() ComputeConfig {
			return cfg
		}),
		fx.Provide(setupComputeService),
	)
}

type ComputeDependencies struct {
	fx.In

	Host              host.Host
	NodeInfoProvider  *routing.NodeInfoProvider
	ApiServer         *publicapi.Server
	StorageProvider   storage.StorageProvider
	ExecutorProvider  executor.ExecutorProvider
	PublisherProvider publisher.PublisherProvider
	Repo              *repo.FsRepo
	// TODO deprecate this
	CleanupManager *system.CleanupManager
}

type ComputeService struct {
	fx.Out

	Compute *Compute
}

func setupComputeService(lc fx.Lifecycle, cfg ComputeConfig, deps ComputeDependencies) (ComputeService, error) {
	ctx, cancel := context.WithCancel(context.TODO())
	node, err := NewComputeNode(
		ctx,
		deps.CleanupManager,
		deps.Host,
		deps.ApiServer,
		cfg,
		deps.StorageProvider,
		deps.ExecutorProvider,
		deps.PublisherProvider,
		deps.Repo,
	)
	if err != nil {
		cancel()
		return ComputeService{}, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting compute module")
			deps.NodeInfoProvider.RegisterComputeInfoProvider(node.computeInfoProvider)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("stopping compute module")
			node.cleanup(ctx)
			cancel()
			return nil
		},
	})

	return ComputeService{
		Compute: node,
	}, nil
}
