package nodefx

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/fx"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity/disk"
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/compute/sensors"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	pkgconfig "github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	compute_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/compute"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	repo_storage "github.com/bacalhau-project/bacalhau/pkg/storage/repo"
)

type ComputeConfig struct {
	Store                        *types.JobStoreConfig
	Providers                    ProvidersConfig
	Capacity                     CapacityTrackerConfig
	DefaultJobExecutionTimeout   time.Duration
	LogRunningExecutionsInterval time.Duration
	// used for calculator, not really a capacity thing as previously used in legacy config
	DefaultJobResourceLimits models.Resources
	// logging server config
	LogStreamBufferSize int // How many messages to buffer in the log stream channel

	// used by node decorator
	JobResourceLimits models.Resources

	// semantic bid strat config
	JobSelectionPolicy node.JobSelectionPolicy
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	// semantic bid strat config
	MinJobExecutionTimeout time.Duration
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	// semantic bid strat config
	MaxJobExecutionTimeout time.Duration
	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	// semantic bid strat config
	JobExecutionTimeoutClientIDBypassList []string

	// used for label providing.. listed as capacity config :/ none of this shit make any sense
	TotalResourceLimits models.Resources
}

type CapacityTrackerConfig struct {
	TotalResourceLimits models.Resources
	QueueResourceLimits models.Resources
}

type ProvidersConfig struct {
	Executor  map[string][]byte
	Storage   map[string][]byte
	Publisher map[string][]byte
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
	DebugInfoProviders []model.DebugInfoProvider
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

	nodeInfoDecorator  models.NodeInfoDecorator
	autoLabelsProvider models.LabelsProvider
	debugInfoProviders []model.DebugInfoProvider
}

func (n *ComputeNode) DebugInfoProviders() []model.DebugInfoProvider {
	return n.debugInfoProviders
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
		nodeInfoDecorator:  p.NodeInfoDecorator,
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

func Compute() fx.Option {
	// TODO do this as a part of starting the node
	/*

		startup := compute.NewStartup(executionStore, bufferRunner)
		startupErr := startup.Execute(ctx)
		if startupErr != nil {
			return nil, fmt.Errorf("failed to execute compute node startup tasks: %s", startupErr)
		}
	*/
	// TODO do this after the compute and requester nodes have been created
	// there is a similar thing on the requester nodes, these are all collected and
	// used in the shared api endpoint
	/*
		// register debug info providers for the /debug endpoint
		debugInfoProviders := []model.DebugInfoProvider{
			runningInfoProvider,
			sensors.NewCompletedJobs(executionStore),
		}
	*/
	return fx.Options(
		fx.Provide(func(cfg *NodeConfig) *ComputeConfig {
			return cfg.ComputeConfig
		}),
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
		fx.Provide(Executor),
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
		fx.Provide(DebugInfoProvider),
		fx.Provide(BaseEndpoint),
		fx.Provide(LabelsProvider),
		fx.Provide(ManagementClient),

		fx.Invoke(InitLoggingSensor),
		fx.Invoke(RegisterComputeEndpoint),
	)
}

type CapacityTrackerResult struct {
	fx.Out

	Running  capacity.Tracker `name:"running"`
	Enqueued capacity.Tracker `name:"enqueued"`
}

func CapacityTrackers(cfg *ComputeConfig) (CapacityTrackerResult, error) {
	runningCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: cfg.Capacity.TotalResourceLimits,
	})
	enqueuedCapacityTracker := capacity.NewLocalTracker(capacity.LocalTrackerParams{
		MaxCapacity: cfg.Capacity.QueueResourceLimits,
	})

	return CapacityTrackerResult{
		Running:  runningCapacityTracker,
		Enqueued: enqueuedCapacityTracker,
	}, nil
}

type ExecutorParams struct {
	fx.In

	ComputeCallback compute.Callback
	Store           store.ExecutionStore
	Storages        storage.StorageProvider
	Executors       executor.ExecutorProvider
	Publisher       publisher.PublisherProvider
	ResultsPath     *compute.ResultsPath
	Running         capacity.Tracker `name:"running"`
	Enqueued        capacity.Tracker `name:"enqueued"`
}

func Executor(cfg *NodeConfig, p ExecutorParams) (*compute.ExecutorBuffer, error) {
	baseExecutor := compute.NewBaseExecutor(compute.BaseExecutorParams{
		ID:       cfg.NodeID,
		Callback: p.ComputeCallback,
		Store:    p.Store,
		// TODO this needs to be a field
		StorageDirectory: pkgconfig.GetStoragePath(),
		Storages:         p.Storages,
		Executors:        p.Executors,
		Publishers:       p.Publisher,
		// TODO this shouldn't even be a thing!!!!
		FailureInjectionConfig: model.FailureInjectionComputeConfig{IsBadActor: false},
		ResultsPath:            *p.ResultsPath,
	})

	bufferRunner := compute.NewExecutorBuffer(compute.ExecutorBufferParams{
		ID:                         cfg.NodeID,
		DelegateExecutor:           baseExecutor,
		Callback:                   p.ComputeCallback,
		RunningCapacityTracker:     p.Running,
		EnqueuedCapacityTracker:    p.Enqueued,
		DefaultJobExecutionTimeout: cfg.ComputeConfig.DefaultJobExecutionTimeout,
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

func InitLoggingSensor(lc fx.Lifecycle, cfg *ComputeConfig, provider *sensors.RunningExecutionsInfoProvider) error {
	if cfg.LogRunningExecutionsInterval > 0 {
		loggingSensor := sensors.NewLoggingSensor(sensors.LoggingSensorParams{
			InfoProvider: provider,
			Interval:     cfg.LogRunningExecutionsInterval,
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

func UsageCalculator(cfg *ComputeConfig, storages storage.StorageProvider) (capacity.UsageCalculator, error) {
	// endpoint/frontend
	return capacity.NewChainedUsageCalculator(capacity.ChainedUsageCalculatorParams{
		Calculators: []capacity.UsageCalculator{
			capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{
				Defaults: cfg.DefaultJobResourceLimits,
			}),
			disk.NewDiskUsageCalculator(disk.DiskUsageCalculatorParams{
				Storages: storages,
			}),
		},
	}), nil
}

func LoggingServer(cfg *ComputeConfig, executors executor.ExecutorProvider, s store.ExecutionStore) (*logstream.Server, error) {
	return logstream.NewServer(logstream.ServerParams{
		ExecutionStore: s,
		Executors:      executors,
		Buffer:         cfg.LogStreamBufferSize,
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

func NodeDecorator(cfg *ComputeConfig, p NodeDecoratorParams) (models.NodeInfoDecorator, error) {
	return compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:          p.Executors,
		Publisher:          p.Publisher,
		Storages:           p.Storages,
		CapacityTracker:    p.Running,
		ExecutorBuffer:     p.Executor,
		MaxJobRequirements: cfg.JobResourceLimits,
	}), nil
}

type SemanticStrategiesParams struct {
	fx.In

	Storages  storage.StorageProvider
	Executors executor.ExecutorProvider
	Publisher publisher.PublisherProvider
}

// TODO allow these to be configured
func DefaultSemanticStrategies(cfg *ComputeConfig, p SemanticStrategiesParams) ([]bidstrategy.SemanticBidStrategy, error) {
	return []bidstrategy.SemanticBidStrategy{
		semantic.NewNetworkingStrategy(cfg.JobSelectionPolicy.AcceptNetworkedJobs),
		semantic.NewTimeoutStrategy(semantic.TimeoutStrategyParams{
			MaxJobExecutionTimeout:                cfg.MaxJobExecutionTimeout,
			MinJobExecutionTimeout:                cfg.MinJobExecutionTimeout,
			JobExecutionTimeoutClientIDBypassList: cfg.JobExecutionTimeoutClientIDBypassList,
		}),
		semantic.NewStatelessJobStrategy(semantic.StatelessJobStrategyParams{
			RejectStatelessJobs: cfg.JobSelectionPolicy.RejectStatelessJobs,
		}),
		semantic.NewProviderInstalledStrategy(
			p.Publisher,
			func(j *models.Job) string { return j.Task().Publisher.Type },
		),
		semantic.NewStorageInstalledBidStrategy(p.Storages),
		semantic.NewInputLocalityStrategy(semantic.InputLocalityStrategyParams{
			Locality: cfg.JobSelectionPolicy.Locality,
			Storages: p.Storages,
		}),
		semantic.NewExternalCommandStrategy(semantic.ExternalCommandStrategyParams{
			Command: cfg.JobSelectionPolicy.ProbeExec,
		}),
		semantic.NewExternalHTTPStrategy(semantic.ExternalHTTPStrategyParams{
			URL: cfg.JobSelectionPolicy.ProbeHTTP,
		}),
		executor_util.NewExecutorSpecificBidStrategy(p.Executors),
	}, nil
}

type ResourceStrategiesParams struct {
	fx.In

	Executors executor.ExecutorProvider
	Running   capacity.Tracker `name:"running"`
	Enqueued  capacity.Tracker `name:"enqueued"`
}

// TODO allow these to be configured
func DefaultResourceStrategies(cfg *ComputeConfig, p ResourceStrategiesParams) ([]bidstrategy.ResourceBidStrategy, error) {
	return []bidstrategy.ResourceBidStrategy{
		resource.NewMaxCapacityStrategy(resource.MaxCapacityStrategyParams{
			MaxJobRequirements: cfg.JobResourceLimits,
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

	Server     *Server
	Executor   *compute.ExecutorBuffer
	Store      store.ExecutionStore
	Callback   compute.Callback
	Calculator capacity.UsageCalculator

	SemanticStrategies []bidstrategy.SemanticBidStrategy `group:"semantic_strategies"`
	ResourceStrategies []bidstrategy.ResourceBidStrategy `group:"resource_strategies"`
}

func Bidder(cfg *NodeConfig, p BidderParams) (compute.Bidder, error) {
	return compute.NewBidder(compute.BidderParams{
		NodeID:           cfg.NodeID,
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

	Store      store.ExecutionStore
	Calculator capacity.UsageCalculator
	Bidder     compute.Bidder
	Executor   *compute.ExecutorBuffer
	LogServer  *logstream.Server
}

func BaseEndpoint(cfg *NodeConfig, p BaseEndpointParams) (compute.Endpoint, error) {
	return compute.NewBaseEndpoint(compute.BaseEndpointParams{
		ID:              cfg.NodeID,
		ExecutionStore:  p.Store,
		UsageCalculator: p.Calculator,
		Bidder:          p.Bidder,
		Executor:        p.Executor,
		LogServer:       p.LogServer,
	}), nil
}

func DebugInfoProvider(
	sensor *sensors.RunningExecutionsInfoProvider,
	store store.ExecutionStore,
) ([]model.DebugInfoProvider, error) {
	// register debug info providers for the /debug endpoint
	return []model.DebugInfoProvider{
		sensor,
		sensors.NewCompletedJobs(store),
	}, nil
}

func RegisterComputeEndpoint(
	router *echo.Echo,
	bidder compute.Bidder,
	store store.ExecutionStore,
	debugProviders []model.DebugInfoProvider,
) error {
	// register compute public http apis
	compute_endpoint.NewEndpoint(compute_endpoint.EndpointParams{
		Router:             router,
		Bidder:             bidder,
		Store:              store,
		DebugInfoProviders: debugProviders,
	})
	return nil
}

func LabelsProvider(cfg *NodeConfig) (models.LabelsProvider, error) {
	// Compute Node labels
	return models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: cfg.Labels},
		&node.RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(cfg.ComputeConfig.TotalResourceLimits),
		repo_storage.NewLabelsProvider(),
	), nil
}

type ManagementClientParams struct {
	fx.In

	Transport     *nats_transport.NATSTransport
	NodeDecorator models.NodeInfoDecorator
	Running       capacity.Tracker `name:"running"`
	LabelProvider models.LabelsProvider
}

func ManagementClient(
	lc fx.Lifecycle,
	cfg *NodeConfig,
	p ManagementClientParams,
) (*compute.ManagementClient, error) {

	// TODO: Make the registration lock folder a config option so that we have it
	// available and don't have to depend on getting the repo folder.
	repo, _ := pkgconfig.Get[string]("repo")
	regFilename := fmt.Sprintf("%s.registration.lock", cfg.NodeID)
	regFilename = filepath.Join(repo, pkgconfig.ComputeStorePath, regFilename)

	// Set up the management client which will attempt to register this node
	// with the requester node, and then if successful will send regular node
	// info updates.
	managementClient := compute.NewManagementClient(compute.ManagementClientParams{
		NodeID:               cfg.NodeID,
		LabelsProvider:       p.LabelProvider,
		ManagementProxy:      p.Transport.ManagementProxy(),
		NodeInfoDecorator:    p.NodeDecorator,
		RegistrationFilePath: regFilename,
		ResourceTracker:      p.Running,
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
