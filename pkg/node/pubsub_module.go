package node

import (
	"context"
	"fmt"
	"time"

	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/rs/zerolog/log"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

func NewPubSubService(cfg PubSubConfig) fx.Option {
	return fx.Module("pubsub",
		fx.Provide(func() PubSubConfig {
			return cfg
		}),
		// TODO this might need to be provide then an invoke.
		fx.Provide(makePubSubService),
		fx.Invoke(invokePubSubService),
	)
}

type PubSubConfig struct {
	Gossipsub          GossipSubConfig
	NodeInfoPubSub     NodeInfoPubSubConfig
	NodeInfoSubscriber NodeInfoSubscriberConfig
	NodeInfoProvider   NodeInfoProviderConfig
	NodeInfoPublisher  NodeInfoPublisherConfig
}

type PubSubDependencies struct {
	fx.In

	Host  host.Host
	Store routing.NodeInfoStore
}

type PubSubService struct {
	fx.Out

	GossipSub          *libp2p_pubsub.PubSub
	NodeInfoPubSub     *libp2p.PubSub[models.NodeInfo]
	NodeInfoSubscriber *pubsub.ChainedSubscriber[models.NodeInfo]
	NodeInfoProvider   *routing.NodeInfoProvider
	NodeInfoPublisher  *routing.NodeInfoPublisher
}

func makePubSubService(cfg PubSubConfig, deps PubSubDependencies) (PubSubService, error) {
	basicHost, ok := deps.Host.(*basichost.BasicHost)
	if !ok {
		return PubSubService{}, fmt.Errorf("creating node info subscriber host is not a basic host")
	}
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:            basicHost,
		IdentityService: basicHost.IDService(),
		Labels:          cfg.NodeInfoProvider.Labels,
		BacalhauVersion: cfg.NodeInfoProvider.Version,
	})

	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](cfg.NodeInfoSubscriber.IgnoreErrors)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](deps.Store.Add))

	tracer, err := libp2p_pubsub.NewJSONTracer(cfg.Gossipsub.TracerPath)
	if err != nil {
		return PubSubService{}, err
	}

	pgParams := libp2p_pubsub.NewPeerGaterParams(
		cfg.Gossipsub.Threshold,
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.GlobalDecay),
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.SourceDecay),
	)

	gossipSub, err := libp2p_pubsub.NewGossipSub(
		// really annoying this constructor expects a context and doesn't have a Start method.
		context.TODO(),
		deps.Host,
		libp2p_pubsub.WithPeerExchange(cfg.Gossipsub.PeerExchange),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
	if err != nil {
		return PubSubService{}, err
	}

	nodeInfoPubSub, err := libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:        deps.Host,
		PubSub:      gossipSub,
		TopicName:   cfg.NodeInfoPubSub.Topic,
		IgnoreLocal: cfg.NodeInfoPubSub.IgnoreLocal,
	})
	if err != nil {
		return PubSubService{}, fmt.Errorf("creating node info pubsub: %w", err)
	}

	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           nodeInfoPubSub,
		NodeInfoProvider: nodeInfoProvider,
		IntervalConfig:   cfg.NodeInfoPublisher.Interval,
	})

	return PubSubService{
		GossipSub:          gossipSub,
		NodeInfoPubSub:     nodeInfoPubSub,
		NodeInfoSubscriber: nodeInfoSubscriber,
		NodeInfoProvider:   nodeInfoProvider,
		NodeInfoPublisher:  nodeInfoPublisher,
	}, nil
}

func invokePubSubService(lc fx.Lifecycle, service PubSubService) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting pubsub module")
			if err := service.NodeInfoPubSub.Subscribe(ctx, service.NodeInfoSubscriber); err != nil {
				return fmt.Errorf("node info pubsub failed to subscribe to node info: %w", err)
			}
			service.NodeInfoPublisher.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("stopping pubsub module")
			service.NodeInfoPublisher.Stop(ctx)
			if err := service.NodeInfoPubSub.Close(ctx); err != nil {
				return fmt.Errorf("closing node into pub sub: %w", err)
			}
			return nil
		},
	})
}

func setupPubSubService(lc fx.Lifecycle, cfg PubSubConfig, deps PubSubDependencies) (PubSubService, error) {
	basicHost, ok := deps.Host.(*basichost.BasicHost)
	if !ok {
		return PubSubService{}, fmt.Errorf("creating node info subscriber host is not a basic host")
	}
	nodeInfoProvider := routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:            basicHost,
		IdentityService: basicHost.IDService(),
		Labels:          cfg.NodeInfoProvider.Labels,
		BacalhauVersion: cfg.NodeInfoProvider.Version,
	})

	nodeInfoSubscriber := pubsub.NewChainedSubscriber[models.NodeInfo](cfg.NodeInfoSubscriber.IgnoreErrors)

	tracer, err := libp2p_pubsub.NewJSONTracer(cfg.Gossipsub.TracerPath)
	if err != nil {
		return PubSubService{}, err
	}

	pgParams := libp2p_pubsub.NewPeerGaterParams(
		cfg.Gossipsub.Threshold,
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.GlobalDecay),
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.SourceDecay),
	)

	gossipSub, err := libp2p_pubsub.NewGossipSub(
		// really annoying this constructor expects a context and doesn't have a Start method.
		context.TODO(),
		deps.Host,
		libp2p_pubsub.WithPeerExchange(cfg.Gossipsub.PeerExchange),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
	if err != nil {
		return PubSubService{}, err
	}

	nodeInfoPubSub, err := libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:        deps.Host,
		PubSub:      gossipSub,
		TopicName:   cfg.NodeInfoPubSub.Topic,
		IgnoreLocal: cfg.NodeInfoPubSub.IgnoreLocal,
	})
	if err != nil {
		return PubSubService{}, fmt.Errorf("creating node info pubsub: %w", err)
	}

	nodeInfoPublisher := routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           nodeInfoPubSub,
		NodeInfoProvider: nodeInfoProvider,
		IntervalConfig:   cfg.NodeInfoPublisher.Interval,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting pubsub module")
			nodeInfoSubscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](deps.Store.Add))
			if err := nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber); err != nil {
				return fmt.Errorf("node info pubsub failed to subscribe to node info: %w", err)
			}
			nodeInfoPublisher.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("stopping pubsub module")
			nodeInfoPublisher.Stop(ctx)
			if err := nodeInfoPubSub.Close(ctx); err != nil {
				return fmt.Errorf("closing node into pub sub: %w", err)
			}
			return nil
		},
	})

	return PubSubService{
		GossipSub:          gossipSub,
		NodeInfoPubSub:     nodeInfoPubSub,
		NodeInfoSubscriber: nodeInfoSubscriber,
		NodeInfoProvider:   nodeInfoProvider,
		NodeInfoPublisher:  nodeInfoPublisher,
	}, nil

}

func newLibp2pGossipSub(ctx context.Context, h host.Host, cfg GossipSubConfig) (*libp2p_pubsub.PubSub, error) {
	tracer, err := libp2p_pubsub.NewJSONTracer(cfg.TracerPath)
	if err != nil {
		return nil, err
	}

	pgParams := libp2p_pubsub.NewPeerGaterParams(
		cfg.Threshold,
		libp2p_pubsub.ScoreParameterDecay(cfg.GlobalDecay),
		libp2p_pubsub.ScoreParameterDecay(cfg.SourceDecay),
	)

	gossipSub, err := libp2p_pubsub.NewGossipSub(
		// really annoying this constructor expects a context and doesn't have a Start method.
		ctx,
		h,
		libp2p_pubsub.WithPeerExchange(cfg.PeerExchange),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
	if err != nil {
		return nil, err
	}
	return gossipSub, nil
}

type GossipSubConfig struct {
	TracerPath   string
	Threshold    float64
	GlobalDecay  time.Duration
	SourceDecay  time.Duration
	PeerExchange bool
}

type NodeInfoPubSubConfig struct {
	Topic       string
	IgnoreLocal bool
}

type NodeInfoSubscriberConfig struct {
	IgnoreErrors bool
}

type NodeInfoProviderConfig struct {
	Labels  map[string]string
	Version models.BuildVersionInfo
}

type NodeInfoPublisherConfig struct {
	Interval routing.NodeInfoPublisherIntervalConfig
}
