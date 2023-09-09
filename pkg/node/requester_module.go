package node

import (
	"context"

	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func NewRequesterService(cfg RequesterConfig) fx.Option {
	return fx.Module("requester",
		fx.Provide(func() RequesterConfig {
			return cfg
		}),
		fx.Provide(setupRequesterService),
	)
}

type RequesterDependencies struct {
	fx.In

	Host            host.Host
	ApiServer       *publicapi.Server
	StorageProvider storage.StorageProvider
	Store           routing.NodeInfoStore
	Gossipsub       *libp2p_pubsub.PubSub
	Repo            *repo.FsRepo
}

type RequesterService struct {
	fx.Out

	Requester *Requester
}

func setupRequesterService(lc fx.Lifecycle, cfg RequesterConfig, deps RequesterDependencies) (RequesterService, error) {
	ctx, cancel := context.WithCancel(context.TODO())
	node, err := NewRequesterNode(
		ctx,
		deps.Host,
		deps.ApiServer,
		cfg,
		deps.StorageProvider,
		deps.Store,
		deps.Gossipsub,
		deps.Repo,
	)
	if err != nil {
		cancel()
		return RequesterService{}, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting requester service")
			// TODO something here I think?
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("stopping requester service")
			node.cleanup(ctx)
			cancel()
			return nil
		},
	})

	return RequesterService{
		Requester: node,
	}, nil
}
