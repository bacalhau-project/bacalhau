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
		fx.Provide(makeGossipSub),
		fx.Provide(makeNodeInfoPubSub),
		fx.Provide(makeNodeInfoProvider),
		fx.Provide(makeNodeInfoSubscriber),
		fx.Provide(makeNodeInfoPublisher),
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

func makeGossipSub(cfg PubSubConfig, host host.Host) (*libp2p_pubsub.PubSub, error) {
	tracer, err := libp2p_pubsub.NewJSONTracer(cfg.Gossipsub.TracerPath)
	if err != nil {
		return nil, err
	}
	pgParams := libp2p_pubsub.NewPeerGaterParams(
		cfg.Gossipsub.Threshold,
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.GlobalDecay),
		libp2p_pubsub.ScoreParameterDecay(cfg.Gossipsub.SourceDecay),
	)
	return libp2p_pubsub.NewGossipSub(
		context.TODO(),
		host,
		libp2p_pubsub.WithPeerExchange(cfg.Gossipsub.PeerExchange),
		libp2p_pubsub.WithPeerGater(pgParams),
		libp2p_pubsub.WithEventTracer(tracer),
	)
}

func makeNodeInfoProvider(
	cfg PubSubConfig,
	host host.Host,
) (*routing.NodeInfoProvider, error) {
	basicHost, ok := host.(*basichost.BasicHost)
	if !ok {
		return nil, fmt.Errorf("host is not a basic host")
	}
	return routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:            basicHost,
		IdentityService: basicHost.IDService(),
		Labels:          cfg.NodeInfoProvider.Labels,
		BacalhauVersion: cfg.NodeInfoProvider.Version,
	}), nil
}

func makeNodeInfoSubscriber(
	cfg PubSubConfig,
	store routing.NodeInfoStore,
) (*pubsub.ChainedSubscriber[models.NodeInfo], error) {
	subscriber := pubsub.NewChainedSubscriber[models.NodeInfo](cfg.NodeInfoSubscriber.IgnoreErrors)
	subscriber.Add(pubsub.SubscriberFunc[models.NodeInfo](store.Add))
	return subscriber, nil
}

func makeNodeInfoPubSub(
	cfg PubSubConfig,
	host host.Host,
	gossipSub *libp2p_pubsub.PubSub,
) (*libp2p.PubSub[models.NodeInfo], error) {
	return libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:        host,
		PubSub:      gossipSub,
		TopicName:   cfg.NodeInfoPubSub.Topic,
		IgnoreLocal: cfg.NodeInfoPubSub.IgnoreLocal,
	})
}

func makeNodeInfoPublisher(
	cfg PubSubConfig,
	pubsub *libp2p.PubSub[models.NodeInfo],
	provider *routing.NodeInfoProvider,
) (*routing.NodeInfoPublisher, error) {
	return routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           pubsub,
		NodeInfoProvider: provider,
		IntervalConfig:   cfg.NodeInfoPublisher.Interval,
	}), nil
}

func invokePubSubService(
	lc fx.Lifecycle,
	nodeInfoPubSub *libp2p.PubSub[models.NodeInfo],
	nodeInfoSubscriber *pubsub.ChainedSubscriber[models.NodeInfo],
	nodeInfoPublisher *routing.NodeInfoPublisher,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			log.Ctx(ctx).Info().Msg("starting pubsub module")
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
