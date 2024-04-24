package compute

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/BTBurke/k8sresource"
	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	pkgerrors "github.com/pkg/errors"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	compute_system "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/node/heartbeat"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	repo_storage "github.com/bacalhau-project/bacalhau/pkg/storage/repo"
)

var Module = fx.Module("compute",
	fx.Provide(LoadConfig),
	fx.Decorate(DecorateCapacityConfig),
	fx.Provide(NewComputeNode),
	fx.Provide(ExecutionStore),
	fx.Provide(StorageProviders),
	fx.Provide(ExecutorProviders),
	fx.Provide(PublisherProviders),
	fx.Provide(CapacityTrackers),
	fx.Provide(func(lc fx.Lifecycle) (*compute.ResultsPath, error) {
		resultPath, err := compute.NewResultsPath()
		if err != nil {
			return nil, fmt.Errorf("creating compute node result path: %w", err)
		}
		lc.Append(fx.Hook{OnStop: func(ctx context.Context) error {
			return resultPath.Close()
		}})
		return resultPath, nil
	}),
	fx.Provide(BaseExecutor),
	fx.Provide(BufferedExecutor),
	fx.Provide(RunningExecutionsInfoProvider),
	fx.Provide(UsageCalculator),
	fx.Provide(LoggingServer),
	fx.Provide(NodeDecorator),
	// could be a bidder module
	fx.Provide(
		fx.Annotate(
			DefaultSemanticStrategies,
			fx.ResultTags(`group:"semantic_strategies"`),
		),
	),
	fx.Provide(
		fx.Annotate(
			DefaultResourceStrategies,
			fx.ResultTags(`group:"resource_strategies"`),
		),
	),
	fx.Provide(Bidder),
	fx.Provide(
		fx.Annotate(
			DebugInfoProviders,
			fx.ResultTags(`name:"compute_debug_providers"`),
		),
	),
	fx.Provide(BaseEndpoint),
	fx.Provide(LabelsProvider),
	fx.Provide(ManagementClient),
	fx.Provide(HeartbeatClient),

	fx.Invoke(InitLoggingSensor),
	fx.Invoke(RegisterComputeEndpoint),
)

type ConfigResult struct {
	fx.Out

	Capacity           *types.CapacityConfig
	ExecutionStore     types.JobStoreConfig `name:"execution_store_config"`
	JobTimeouts        types.JobTimeoutConfig
	Queue              types.QueueConfig
	LoggingSensor      types.LoggingConfig
	ManifestCache      types.DockerCacheConfig
	DockerCredentials  types.DockerCredentialsConfig
	LogStreamConfig    types.LogStreamConfig
	LocalPublisher     types.LocalPublisherConfig
	Labels             types.LabelsConfig
	Executor           types.ExecutorConfig
	BufferedExecutor   types.BufferedExecutorConfig
	JobSelection       types.JobSelectionPolicyConfig
	StorageProviders   types.StorageProvidersConfig
	ExecutorProviders  types.ExecutorProvidersConfig
	PublisherProviders types.PublisherProvidersConfig
	ControlPlane       types.ComputeControlPlaneConfig
}

// TODO provide the CapacityProvider as a param to this and remove the dep on the config
func DecorateCapacityConfig(cfg *types.CapacityConfig, c *config.Config) (*types.CapacityConfig, error) {
	// decorate/modify any config with default values if none were provided.
	// TODO write decorators for the rest of the config in need of defaults
	if cfg.DefaultJobResourceLimits.CPU == "" {
		cfg.DefaultJobResourceLimits.CPU = "100m"
	}
	if cfg.DefaultJobResourceLimits.Memory == "" {
		cfg.DefaultJobResourceLimits.Memory = "100Mi"
	}
	strgPath, found := c.GetString(types.NodeComputeStoragePath)
	if !found {
		return nil, fmt.Errorf("%s not configured", types.NodeComputeStoragePath)
	}
	physicalResourcesProvider := compute_system.NewPhysicalCapacityProvider(strgPath)
	physicalResources, err := physicalResourcesProvider.GetAvailableCapacity(context.TODO())
	if err != nil {
		return nil, err
	}
	if cfg.TotalResourceLimits.CPU == "" {
		cpu := k8sresource.NewCPUFromFloat(physicalResources.CPU)
		cfg.TotalResourceLimits.CPU = cpu.ToString()

	}
	if cfg.TotalResourceLimits.Memory == "" {
		ram := humanize.Bytes(physicalResources.Memory)
		cfg.TotalResourceLimits.Memory = ram

	}
	if cfg.TotalResourceLimits.GPU == "" {
		cfg.TotalResourceLimits.GPU = fmt.Sprintf("%d", physicalResources.GPU)
	}
	if cfg.TotalResourceLimits.Disk == "" {
		cfg.TotalResourceLimits.Disk = fmt.Sprintf("%d", physicalResources.Disk)
	}
	return cfg, nil

}

func LoadConfig(c *config.Config) (ConfigResult, error) {
	var cfg types.ComputeConfig
	if err := c.ForKey(types.NodeCompute, &cfg); err != nil {
		return ConfigResult{}, err
	}

	return ConfigResult{
		Capacity:           &cfg.Capacity,
		ExecutionStore:     cfg.ExecutionStore,
		JobTimeouts:        cfg.JobTimeouts,
		Queue:              cfg.Queue,
		LoggingSensor:      cfg.Logging,
		ManifestCache:      cfg.ManifestCache,
		LogStreamConfig:    cfg.LogStreamConfig,
		LocalPublisher:     cfg.LocalPublisher,
		Labels:             cfg.Labels,
		Executor:           cfg.Executor,
		BufferedExecutor:   cfg.BufferedExecutor,
		JobSelection:       cfg.JobSelection,
		StorageProviders:   cfg.StorageProviders,
		ExecutorProviders:  cfg.ExecutorProviders,
		PublisherProviders: cfg.PublisherProviders,
		DockerCredentials:  cfg.DockerCredentials,
		ControlPlane:       cfg.ControlPlaneSettings,
	}, nil
}

type ComputeNodeParams struct {
	fx.In

	ExecutionStore          store.ExecutionStore
	Storage                 storage.StorageProvider
	Executors               executor.ExecutorProvider
	Publisher               publisher.PublisherProvider
	Executor                *compute.ExecutorBuffer
	Bidder                  compute.Bidder
	LocalEndpoint           compute.Endpoint
	ManagementClient        *compute.ManagementClient
	RunningCapacityTraker   capacity.Tracker `name:"running"`
	EnqueuedCapacityTracker capacity.Tracker `name:"enqueued"`

	NodeInfoDecorator  models.NodeInfoDecorator
	AutoLabelsProvider models.LabelsProvider
	DebugInfoProviders []model.DebugInfoProvider `name:"compute_debug_providers"`
}

type ComputeNode struct {
	ExecutionStore     store.ExecutionStore
	StorageProviders   storage.StorageProvider
	ExecutorProviders  executor.ExecutorProvider
	PublisherProviders publisher.PublisherProvider
	Executor           *compute.ExecutorBuffer
	Bidder             compute.Bidder
	ManagementClient   *compute.ManagementClient
	LocalEndpoint      compute.Endpoint

	NodeInfoDecorator  models.NodeInfoDecorator
	autoLabelsProvider models.LabelsProvider
	debugInfoProviders []model.DebugInfoProvider
}

func NewComputeNode(lc fx.Lifecycle, p ComputeNodeParams) *ComputeNode {
	n := &ComputeNode{
		LocalEndpoint:      p.LocalEndpoint,
		ExecutionStore:     p.ExecutionStore,
		StorageProviders:   p.Storage,
		ExecutorProviders:  p.Executors,
		PublisherProviders: p.Publisher,
		Executor:           p.Executor,
		Bidder:             p.Bidder,
		ManagementClient:   p.ManagementClient,
		NodeInfoDecorator:  p.NodeInfoDecorator,
		autoLabelsProvider: p.AutoLabelsProvider,
		debugInfoProviders: p.DebugInfoProviders,
	}

	// TODO this is suppoed to gate node construction, but that doesn't really flow.
	lc.Append(fx.Hook{OnStart: func(ctx context.Context) error {
		if err := compute.NewStartup(n.ExecutionStore, n.Executor).Execute(ctx); err != nil {
			return fmt.Errorf("failed to execute compute node startup tasks: %w", err)
		}
		return nil
	}})

	return n
}

type CapacityTrackerResult struct {
	fx.Out

	Running  capacity.Tracker `name:"running"`
	Enqueued capacity.Tracker `name:"enqueued"`
}

func CapacityTrackers(cfg *types.CapacityConfig) (CapacityTrackerResult, error) {
	totalResources, err := cfg.TotalResourceLimits.ToResources()
	if err != nil {
		return CapacityTrackerResult{}, err
	}
	queuedResources, err := cfg.TotalResourceLimits.ToResources()
	if err != nil {
		return CapacityTrackerResult{}, err
	}
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: *totalResources,
	})
	enqueuedCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: *queuedResources,
	})

	return CapacityTrackerResult{
		Running:  runningCapacityTracker,
		Enqueued: enqueuedCapacityTracker,
	}, nil
}

type BufferedExecutorParams struct {
	fx.In

	NodeID          types.NodeID
	Config          types.BufferedExecutorConfig
	ComputeCallback compute.Callback
	Store           store.ExecutionStore
	Storages        storage.StorageProvider
	Executors       executor.ExecutorProvider
	Publisher       publisher.PublisherProvider
	ResultsPath     *compute.ResultsPath
	Running         capacity.Tracker `name:"running"`
	Enqueued        capacity.Tracker `name:"enqueued"`
	BaseExecutor    *compute.BaseExecutor
}

type BaseExecutorParams struct {
	fx.In

	NodeID          types.NodeID
	Config          types.ExecutorConfig
	ComputeCallback compute.Callback
	Store           store.ExecutionStore
	Storages        storage.StorageProvider
	Executors       executor.ExecutorProvider
	Publisher       publisher.PublisherProvider
	ResultsPath     *compute.ResultsPath
}

func BaseExecutor(p BaseExecutorParams) (*compute.BaseExecutor, error) {
	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:       string(p.NodeID),
		Callback: p.ComputeCallback,
		Store:    p.Store,
		// TODO this needs to be a field
		StorageDirectory: p.Config.StorageDirectory,
		Storages:         p.Storages,
		Executors:        p.Executors,
		Publishers:       p.Publisher,
		// TODO this shouldn't even be a thing!!!!
		FailureInjectionConfig: model.FailureInjectionComputeConfig{IsBadActor: false},
		// TODO all this to be specified instead of a random tempdir
		ResultsPath: *p.ResultsPath,
	})
	return baseExecutor, nil
}

func BufferedExecutor(p BufferedExecutorParams) (*compute.ExecutorBuffer, error) {
	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                         string(p.NodeID),
		DelegateExecutor:           p.BaseExecutor,
		Callback:                   p.ComputeCallback,
		RunningCapacityTracker:     p.Running,
		EnqueuedCapacityTracker:    p.Enqueued,
		DefaultJobExecutionTimeout: time.Duration(p.Config.DefaultJobExecutionTimeout),
	})

	return bufferRunner, nil
}

func RunningExecutionsInfoProvider(buffer *compute.ExecutorBuffer) (*sensors.RunningExecutionsInfoProvider, error) {
	runningInfoProvider := sensors.NewRunningExecutionsInfoProvider(sensors.RunningExecutionsInfoProviderParams{
		Name:          "ActiveJobs",
		BackendBuffer: buffer,
	})
	return runningInfoProvider, nil
}

func InitLoggingSensor(lc fx.Lifecycle, cfg types.LoggingConfig, provider *sensors.RunningExecutionsInfoProvider) error {
	if cfg.LogRunningExecutionsInterval > 0 {
		loggingSensor := sensors.NewLoggingSensor(sensors.LoggingSensorParams{
			InfoProvider: provider,
			Interval:     time.Duration(cfg.LogRunningExecutionsInterval),
		})
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				// NB: the cancellation of the context when the application closes will stop the sensor
				// TODO: add explicit stop method or rename the `Start` method to run sicne it blocks.
				go loggingSensor.Start(ctx)
				return nil
			},
		})
	}
	return nil
}

func UsageCalculator(cfg *types.CapacityConfig, storages storage.StorageProvider) (capacity.UsageCalculator, error) {
	// endpoint/frontend
	defaults, err := cfg.DefaultJobResourceLimits.ToResources()
	if err != nil {
		return nil, err
	}
	return capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				Defaults: *defaults,
			}),
			disk.NewDiskUsageCalculator(disk.DiskUsageCalculatorParams{
				Storages: storages,
			}),
		},
	}), nil
}

func LoggingServer(cfg types.LogStreamConfig, executors executor.ExecutorProvider, s store.ExecutionStore) (*logstream.Server, error) {
	return logstream.NewServer(logstream.ServerParams{
		ExecutionStore: s,
		Executors:      executors,
		Buffer:         cfg.ChannelBufferSize,
	}), nil
}

type NodeDecoratorParams struct {
	fx.In

	Storages  storage.StorageProvider
	Executors executor.ExecutorProvider
	Publisher publisher.PublisherProvider
	Executor  *compute.ExecutorBuffer
	Running   capacity.Tracker `name:"running"`
}

func NodeDecorator(cfg *types.CapacityConfig, p NodeDecoratorParams) (models.NodeInfoDecorator, error) {
	maxRequirements, err := cfg.JobResourceLimits.ToResources()
	if err != nil {
		return nil, err
	}
	return compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:          p.Executors,
		Publisher:          p.Publisher,
		Storages:           p.Storages,
		CapacityTracker:    p.Running,
		ExecutorBuffer:     p.Executor,
		MaxJobRequirements: *maxRequirements,
	}), nil
}

type SemanticStrategiesParams struct {
	fx.In

	Storages       storage.StorageProvider
	Executors      executor.ExecutorProvider
	Publisher      publisher.PublisherProvider
	PolicyConfig   types.JobSelectionPolicyConfig
	TimeoutsConfig types.JobTimeoutConfig
}

// TODO allow these to be configured
func DefaultSemanticStrategies(p SemanticStrategiesParams) ([]bidstrategy.SemanticBidStrategy, error) {
	return []bidstrategy.SemanticBidStrategy{
		semantic.NewNetworkingStrategy(p.PolicyConfig.Policy.AcceptNetworkedJobs),
		semantic.NewTimeoutStrategy(semantic.TimeoutStrategyParams{
			MaxJobExecutionTimeout:                time.Duration(p.TimeoutsConfig.MaxJobExecutionTimeout),
			MinJobExecutionTimeout:                time.Duration(p.TimeoutsConfig.MinJobExecutionTimeout),
			JobExecutionTimeoutClientIDBypassList: p.TimeoutsConfig.JobExecutionTimeoutClientIDBypassList,
		}),
		semantic.NewStatelessJobStrategy(semantic.StatelessJobStrategyParams{
			RejectStatelessJobs: p.PolicyConfig.Policy.RejectStatelessJobs,
		}),
		semantic.NewProviderInstalledStrategy(
			p.Publisher,
			func(j *models.Job) string { return j.Task().Publisher.Type },
		),
		semantic.NewStorageInstalledBidStrategy(p.Storages),
		semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
			Locality: semantic.JobSelectionDataLocality(p.PolicyConfig.Policy.Locality),
			Storages: p.Storages,
		}),
		semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
			Command: p.PolicyConfig.Policy.ProbeExec,
		}),
		semantic.NewExternalHTTPStrategy(semantic.ExternalHTTPStrategyParams{
			URL: p.PolicyConfig.Policy.ProbeHTTP,
		}),
		executor_util.NewExecutorSpecificBidStrategy(p.Executors),
	}, nil
}

type ResourceStrategiesParams struct {
	fx.In

	Config    *types.CapacityConfig
	Executors executor.ExecutorProvider
	Running   capacity.Tracker `name:"running"`
	Enqueued  capacity.Tracker `name:"enqueued"`
}

// TODO allow these to be configured
func DefaultResourceStrategies(p ResourceStrategiesParams) ([]bidstrategy.ResourceBidStrategy, error) {
	maxRequirements, err := p.Config.JobResourceLimits.ToResources()
	if err != nil {
		return nil, err
	}
	return []bidstrategy.ResourceBidStrategy{
		resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
			MaxJobRequirements: *maxRequirements,
		}),
		resource.NewAvailableCapacityStrategy(resource.AvailableCapacityStrategyParams{
			RunningCapacityTracker:  p.Running,
			EnqueuedCapacityTracker: p.Enqueued,
		}),
		executor_util.NewExecutorSpecificBidStrategy(p.Executors),
	}, nil
}

type BidderParams struct {
	fx.In

	NodeID     types.NodeID
	Server     *publicapi.Server
	Executor   *compute.ExecutorBuffer
	Store      store.ExecutionStore
	Callback   compute.Callback
	Calculator capacity.UsageCalculator

	SemanticStrategies []bidstrategy.SemanticBidStrategy `group:"semantic_strategies"`
	ResourceStrategies []bidstrategy.ResourceBidStrategy `group:"resource_strategies"`
}

func Bidder(p BidderParams) (compute.Bidder, error) {
	return compute.NewBidder(compute.BidderParams{
		NodeID:           string(p.NodeID),
		SemanticStrategy: p.SemanticStrategies,
		ResourceStrategy: p.ResourceStrategies,
		UsageCalculator:  p.Calculator,
		Store:            p.Store,
		Executor:         p.Executor,
		Callback:         p.Callback,
		// TODO this feels wrong, but is copied from existing code.
		GetApproveURL: func() *url.URL { return p.Server.GetURI().JoinPath("/api/v1/compute/approve") },
	}), nil
}

type BaseEndpointParams struct {
	fx.In

	NodeID     types.NodeID
	Store      store.ExecutionStore
	Calculator capacity.UsageCalculator
	Bidder     compute.Bidder
	Executor   *compute.ExecutorBuffer
	LogServer  *logstream.Server
}

func BaseEndpoint(p BaseEndpointParams) (compute.Endpoint, error) {
	return compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              string(p.NodeID),
		ExecutionStore:  p.Store,
		UsageCalculator: p.Calculator,
		Bidder:          p.Bidder,
		Executor:        p.Executor,
		LogServer:       p.LogServer,
	}), nil
}

func DebugInfoProviders(
	sensor *sensors.RunningExecutionsInfoProvider,
	store store.ExecutionStore,
) ([]model.DebugInfoProvider, error) {
	// register debug info providers for the /debug endpoint
	return []model.DebugInfoProvider{
		sensor,
		sensors.NewCompletedJobs(store),
	}, nil
}

type RegisterComputeEndpointParams struct {
	fx.In

	Router         *echo.Echo
	Bidder         compute.Bidder
	Store          store.ExecutionStore
	DebugProviders []model.DebugInfoProvider `name:"compute_debug_providers"`
}

func RegisterComputeEndpoint(p RegisterComputeEndpointParams) error {
	// register compute public http apis
	compute_endpoint.NewEndpoint(compute_endpoint.EndpointParams{
		Router:             p.Router,
		Bidder:             p.Bidder,
		Store:              p.Store,
		DebugInfoProviders: p.DebugProviders,
	})
	return nil
}

func LabelsProvider(labelConifg types.LabelsConfig, capacityConifg *types.CapacityConfig) (models.LabelsProvider, error) {
	// Compute Node labels
	totalLimits, err := capacityConifg.TotalResourceLimits.ToResources()
	if err != nil {
		return nil, err
	}
	return models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: labelConifg.Labels},
		&node.RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(*totalLimits),
		repo_storage.NewLabelsProvider(),
	), nil
}

type ManagementClientParams struct {
	fx.In

	Transport       *nats_transport.NATSTransport
	NodeDecorator   models.NodeInfoDecorator
	Running         capacity.Tracker `name:"running"`
	LabelProvider   models.LabelsProvider
	NodeID          types.NodeID
	Repo            *repo.FsRepo
	Config          types.ComputeControlPlaneConfig
	HeartbeatClient *heartbeat.HeartbeatClient
}

func HeartbeatClient(transport *nats_transport.NATSTransport, nodeID types.NodeID, config types.ComputeControlPlaneConfig) (*heartbeat.HeartbeatClient, error) {
	hbClient, err := heartbeat.NewClient(
		transport.Client().Client,
		string(nodeID),
		config.HeartbeatTopic,
	)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create heartbeat client")
	}

	return hbClient, nil
}

func ManagementClient(
	lc fx.Lifecycle,
	p ManagementClientParams,
) (*compute.ManagementClient, error) {

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	repoPath, err := p.Repo.Path()
	if err != nil {
		return nil, err
	}
	regFilename := fmt.Sprintf("%s.registration.lock", string(p.NodeID))
	regFilename = filepath.Join(repoPath, pkgconfig.ComputeStorePath, regFilename)

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	managementClient := compute.NewManagementClient(compute.ManagementClientParams{
		NodeID:               string(p.NodeID),
		LabelsProvider:       p.LabelProvider,
		ManagementProxy:      p.Transport.ManagementProxy(),
		NodeInfoDecorator:    p.NodeDecorator,
		ResourceTracker:      p.Running,
		RegistrationFilePath: regFilename,
		HeartbeatClient:      p.HeartbeatClient,
		ControlPlaneSettings: p.Config,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := managementClient.RegisterNode(ctx); err != nil {
				return fmt.Errorf("failed to register node with requester: %s", err)
			}
			go managementClient.Start(ctx)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			managementClient.Stop()
			return nil
		},
	})

	return managementClient, nil
}
