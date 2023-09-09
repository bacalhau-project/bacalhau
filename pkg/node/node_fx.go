package node

import (
	"context"
	"time"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/agent"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/shared"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func NewStorageProvider(cfg NodeConfig) (storage.StorageProvider, error) {
	return cfg.DependencyInjector.StorageProvidersFactory.Get(context.TODO(), cfg)
}

func NewPublisherProvider(cfg NodeConfig) (publisher.PublisherProvider, error) {
	return cfg.DependencyInjector.PublishersFactory.Get(context.TODO(), cfg)
}

func NewExecutorProvider(cfg NodeConfig) (executor.ExecutorProvider, error) {
	return cfg.DependencyInjector.ExecutorsFactory.Get(context.TODO(), cfg)
}

func NewAPIServer(cfg NodeConfig, provider *routing.NodeInfoProvider) (*publicapi.Server, error) {
	// timeoutHandler doesn't implement http.Hijacker, so we need to skip it for websocket endpoints
	cfg.APIServerConfig.SkippedTimeoutPaths = append(cfg.APIServerConfig.SkippedTimeoutPaths, []string{
		"/api/v1/requester/websocket/events",
		"/api/v1/requester/logs",
	}...)

	// public http api server
	serverParams := publicapi.ServerParams{
		Router:  echo.New(),
		Address: cfg.HostAddress,
		Port:    cfg.APIPort,
		HostID:  cfg.Host.ID().String(),
		Config:  cfg.APIServerConfig,
	}

	// Only allow autocert for requester nodes
	if cfg.IsRequesterNode {
		serverParams.AutoCertDomain = cfg.RequesterAutoCert
		serverParams.AutoCertCache = cfg.RequesterAutoCertCache
	}

	apiServer, err := publicapi.NewAPIServer(serverParams)
	if err != nil {
		return nil, err
	}

	// TODO(forrest): [correctness] strange we ignore the returns from these constructors.
	shared.NewEndpoint(shared.EndpointParams{
		Router:           apiServer.Router,
		NodeID:           cfg.Host.ID().String(),
		PeerStore:        cfg.Host.Peerstore(),
		NodeInfoProvider: provider,
	})

	agent.NewEndpoint(agent.EndpointParams{
		Router:           apiServer.Router,
		NodeInfoProvider: provider,
	})

	return apiServer, nil
}

type StopFn = func(ctx context.Context) error

func NewNodeWithOptions(ctx context.Context, cfg NodeConfig) (StopFn, error) {
	// yay globals...
	identify.ActivationThresh = 2

	cfg.DependencyInjector = mergeDependencyInjectors(cfg.DependencyInjector, NewStandardNodeDependencyInjector())
	err := mergo.Merge(&cfg.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, err
	}

	nodeInfoPublisherInterval := cfg.NodeInfoPublisherInterval
	if nodeInfoPublisherInterval.IsZero() {
		nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
	}

	app := fx.New(
		fx.Provide(func() host.Host { return cfg.Host }),
		fx.Provide(func() routing.NodeInfoStore {
			return inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{TTL: time.Minute * 10})
		}),
		fx.Provide(func() *system.CleanupManager {
			return cfg.CleanupManager
		}),
		fx.Provide(func() *repo.FsRepo { return cfg.FsRepo }),
		fx.Provide(func() NodeConfig { return cfg }),
		fx.Provide(NewStorageProvider),
		fx.Provide(NewPublisherProvider),
		fx.Provide(NewExecutorProvider),
		fx.Provide(NewAPIServer),
		NewPubSubService(PubSubConfig{
			Gossipsub: GossipSubConfig{
				TracerPath:   config.GetLibp2pTracerPath(),
				Threshold:    0.33,
				GlobalDecay:  2 * time.Minute,
				SourceDecay:  10 * time.Minute,
				PeerExchange: true,
			},
			NodeInfoPubSub: NodeInfoPubSubConfig{
				Topic:       NodeInfoTopic,
				IgnoreLocal: false,
			},
			NodeInfoSubscriber: NodeInfoSubscriberConfig{
				IgnoreErrors: true,
			},
			NodeInfoProvider: NodeInfoProviderConfig{
				Labels:  cfg.Labels,
				Version: *version.Get(),
			},
			NodeInfoPublisher: NodeInfoPublisherConfig{
				Interval: nodeInfoPublisherInterval,
			},
		}),
		NewComputeService(cfg.ComputeConfig),
		NewRequesterService(cfg.RequesterNodeConfig),
		NewNodeService(cfg),
	)
	if err := app.Start(ctx); err != nil {
		return nil, err
	}

	return app.Stop, nil
}
