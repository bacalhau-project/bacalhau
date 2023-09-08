package node

import (
	"context"
	"time"

	"github.com/imdario/mergo"
	"github.com/labstack/echo/v4"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	routedhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
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

func NewNodeInfoStore(cfg NodeConfig) routing.NodeInfoStore {
	// TODO include TTL option in NodeConfig
	return inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		TTL: 10 * time.Minute,
	})
}

func NewNodeInfoSubscriber() *pubsub.ChainedSubscriber[models.NodeInfo] {
	return pubsub.NewChainedSubscriber[models.NodeInfo](true)
}

func SetupNodeInfoSubscriber(lc fx.Lifecycle, sub *pubsub.ChainedSubscriber[models.NodeInfo], store routing.NodeInfoStore) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			sub.Add(pubsub.SubscriberFunc[models.NodeInfo](store.Add))
			return nil
		},
	})
}

func NewNodeInfoPubSub(
	h host.Host,
	gossipSub *libp2p_pubsub.PubSub,
) (*libp2p.PubSub[models.NodeInfo], error) {
	return libp2p.NewPubSub[models.NodeInfo](libp2p.PubSubParams{
		Host:      h,
		TopicName: NodeInfoTopic,
		PubSub:    gossipSub,
	})
}

func SetupNodeInfoPubSub(lc fx.Lifecycle, ps *libp2p.PubSub[models.NodeInfo], sub *pubsub.ChainedSubscriber[models.NodeInfo]) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return ps.Subscribe(ctx, sub)
		},
		OnStop: func(ctx context.Context) error {
			return ps.Close(ctx)
		},
	})
}

func NewNodeInfoProvider(bh *basichost.BasicHost, cfg NodeConfig) *routing.NodeInfoProvider {
	return routing.NewNodeInfoProvider(routing.NodeInfoProviderParams{
		Host:            bh,
		IdentityService: bh.IDService(),
		Labels:          cfg.Labels,
		BacalhauVersion: *version.Get(),
		// TODO at some point we need to register computeInfoProvider
		ComputeInfoProvider: nil,
	})
}

func NewRoutedHost(h host.Host, store routing.NodeInfoStore) *routedhost.RoutedHost {
	return routedhost.Wrap(h, store)
}

func NewAPIServer(cfg NodeConfig) (*publicapi.Server, error) {
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

	return publicapi.NewAPIServer(serverParams)
}

func NewNodeInfoPublisher(pubsub *libp2p.PubSub[models.NodeInfo], provider *routing.NodeInfoProvider, cfg NodeConfig) *routing.NodeInfoPublisher {
	// node info publisher
	nodeInfoPublisherInterval := cfg.NodeInfoPublisherInterval
	if nodeInfoPublisherInterval.IsZero() {
		nodeInfoPublisherInterval = GetNodeInfoPublishConfig()
	}

	// NB(forrest): this must be done last to avoid eager publishing before nodes are constructed
	// TODO(forrest) [fixme] we should fix this to make it less racy in testing
	return routing.NewNodeInfoPublisher(routing.NodeInfoPublisherParams{
		PubSub:           pubsub,
		NodeInfoProvider: provider,
		IntervalConfig:   nodeInfoPublisherInterval,
	})
}

func SetupNodeInfoPublisher(lc fx.Lifecycle, publisher *routing.NodeInfoPublisher) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			publisher.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			publisher.Stop(ctx)
			return nil
		},
	})
}

func NewRequesterNode2(
	h *routedhost.RoutedHost,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	storageProvider storage.StorageProvider,
	store routing.NodeInfoStore,
	gossipsub *libp2p_pubsub.PubSub,
) (*Requester, error) {
	return NewRequesterNode(
		context.TODO(),
		h,
		apiServer,
		cfg.RequesterNodeConfig,
		storageProvider,
		store,
		gossipsub,
		cfg.FsRepo,
	)
}

func SetupRequesterNode(lc fx.Lifecycle, node *Requester) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			node.cleanup(ctx)
			return nil
		},
	})
}

func NewComputeNode2(
	h *routedhost.RoutedHost,
	cfg NodeConfig,
	apiServer *publicapi.Server,
	storageProvider storage.StorageProvider,
	executorProvider executor.ExecutorProvider,
	publisherProvider publisher.PublisherProvider,
) (*Compute, error) {
	return NewComputeNode(
		context.TODO(),
		cfg.CleanupManager,
		h,
		apiServer,
		cfg.ComputeConfig,
		storageProvider,
		executorProvider,
		publisherProvider,
		cfg.FsRepo,
	)
}

func SetupComputeNode(lc fx.Lifecycle, nodeInfoProvider *routing.NodeInfoProvider, node *Compute) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			nodeInfoProvider.RegisterComputeInfoProvider(node.computeInfoProvider)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			node.cleanup(ctx)
			return nil
		},
	})
}

type StopFn = func(ctx context.Context) error

func NewNodeWithOptions(ctx context.Context, cfg NodeConfig) (StopFn, error) {
	cfg.DependencyInjector = mergeDependencyInjectors(cfg.DependencyInjector, NewStandardNodeDependencyInjector())
	err := mergo.Merge(&cfg.APIServerConfig, publicapi.DefaultConfig())
	if err != nil {
		return nil, err
	}
	app := fx.New(
		fx.Provide(func() context.Context { return ctx }),
		fx.Provide(func() NodeConfig { return cfg }),
		fx.Provide(func() host.Host { return cfg.Host }),
		fx.Provide(func() *basichost.BasicHost { return cfg.Host.(*basichost.BasicHost) }),
		fx.Provide(newLibp2pPubSub),
		fx.Provide(NewStorageProvider),
		fx.Provide(NewPublisherProvider),
		fx.Provide(NewExecutorProvider),

		fx.Provide(NewNodeInfoStore),
		fx.Provide(NewNodeInfoSubscriber),
		fx.Provide(NewNodeInfoPubSub),
		fx.Provide(NewNodeInfoProvider),
		fx.Provide(NewRoutedHost),

		fx.Provide(NewAPIServer),
		fx.Provide(NewRequesterNode2),
		fx.Provide(NewComputeNode2),
		fx.Provide(NewNodeInfoPublisher),
		fx.Provide(NewNodeFx),

		fx.Invoke(SetupNodeInfoPubSub),
		fx.Invoke(SetupNodeInfoSubscriber),
		fx.Invoke(SetupNodeInfoPublisher),
		fx.Invoke(SetupComputeNode),
		fx.Invoke(SetupRequesterNode),
		fx.Invoke(SetupNode),
	)
	if err := app.Start(ctx); err != nil {
		return nil, err
	}

	return app.Stop, nil
}

func NewNodeFx(cfg NodeConfig, apiServer *publicapi.Server, compute *Compute, requester *Requester, store routing.NodeInfoStore, h *routedhost.RoutedHost) *Node {
	return &Node{
		APIServer:      apiServer,
		ComputeNode:    compute,
		RequesterNode:  requester,
		NodeInfoStore:  store,
		CleanupManager: cfg.CleanupManager,
		IPFSClient:     cfg.IPFSClient,
		Host:           h,
	}
}

func SetupNode(lc fx.Lifecycle, node *Node) {
	if node.IsComputeNode() && node.IsRequesterNode() {
		node.ComputeNode.RegisterLocalComputeCallback(node.RequesterNode.localCallback)
		node.RequesterNode.RegisterLocalComputeEndpoint(node.ComputeNode.LocalEndpoint)
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return node.Start(ctx)
		},
	})
}
