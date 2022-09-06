package node

import (
	"context"

	computenode "github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/rs/zerolog/log"
)

// Node configuration
type NodeConfig struct {
	IPFSClient           *ipfs.Client
	CleanupManager       *system.CleanupManager
	Transport            transport.Transport
	FilecoinUnsealedPath string
	EstuaryAPIKey        string
	HostAddress          string
	HostID               string
	APIPort              int
	MetricsPort          int
	IsBadActor           bool
	ComputeNodeConfig    computenode.ComputeNodeConfig
	RequesterNodeConfig  requesternode.RequesterNodeConfig
}

// Lazy node dependency injector that generate instances of different
// components on demand and based on the configuration provided.
type NodeDependencyInjector struct {
	StorageProvidersFactory StorageProvidersFactory
	ExecutorsFactory        ExecutorsFactory
	VerifiersFactory        VerifiersFactory
	PublishersFactory       PublishersFactory
}

func NewStandardNodeDepdencyInjector() NodeDependencyInjector {
	return NodeDependencyInjector{
		StorageProvidersFactory: NewStandardStorageProvidersFactory(),
		ExecutorsFactory:        NewStandardExecutorsFactory(),
		VerifiersFactory:        NewStandardVerifiersFactory(),
		PublishersFactory:       NewStandardPublishersFactory(),
	}
}

type Node struct {
	// Visible for testing
	APIServer      *publicapi.APIServer
	ComputeNode    *computenode.ComputeNode
	RequestorNode  *requesternode.RequesterNode
	Controller     *controller.Controller
	Transport      transport.Transport
	CleanupManager *system.CleanupManager
	Executors      map[model.EngineType]executor.Executor
	IPFSClient     *ipfs.Client

	HostID      string
	metricsPort int
}

func (n *Node) StartControllerOnly(ctx context.Context) error {
	if err := n.Controller.Start(ctx); err != nil {
		return err
	}
	return nil
}

func (n *Node) Start(ctx context.Context) error {
	if err := n.StartControllerOnly(ctx); err != nil {
		return err
	}

	go func(ctx context.Context) {
		if err := n.APIServer.ListenAndServe(ctx, n.CleanupManager); err != nil {
			log.Error().Msgf("Api server can't run. Cannot serve client requests!: %v", err)
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := system.ListenAndServeMetrics(ctx, n.CleanupManager, n.metricsPort); err != nil {
			log.Error().Msgf("Cannot serve metrics: %v", err)
		}
	}(ctx)

	return nil
}

func NewStandardNode(
	ctx context.Context,
	config NodeConfig) (*Node, error) {
	return NewNode(ctx, config, NewStandardNodeDepdencyInjector())
}

func NewNode(
	ctx context.Context,
	config NodeConfig,
	injector NodeDependencyInjector) (*Node, error) {
	if config.HostID == "" {
		var err error
		config.HostID, err = config.Transport.HostID(ctx)
		if err != nil {
			return nil, err
		}
	}

	datastore, err := inmemory.NewInMemoryDatastore()
	if err != nil {
		return nil, err
	}

	storageProviders, err := injector.StorageProvidersFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	controller, err := controller.NewController(
		ctx,
		config.CleanupManager,
		datastore,
		config.Transport,
		storageProviders,
	)
	if err != nil {
		return nil, err
	}

	executors, err := injector.ExecutorsFactory.Get(ctx, config)
	if err != nil {
		return nil, err
	}

	verifiers, err := injector.VerifiersFactory.Get(ctx, config, controller)
	if err != nil {
		return nil, err
	}

	publishers, err := injector.PublishersFactory.Get(ctx, config, controller)
	if err != nil {
		return nil, err
	}

	requesterNode, err := requesternode.NewRequesterNode(
		ctx,
		config.CleanupManager,
		controller,
		verifiers,
		config.RequesterNodeConfig,
	)
	if err != nil {
		return nil, err
	}
	computeNode, err := computenode.NewComputeNode(
		ctx,
		config.CleanupManager,
		controller,
		executors,
		verifiers,
		publishers,
		config.ComputeNodeConfig,
	)
	if err != nil {
		return nil, err
	}

	apiServer := publicapi.NewServer(
		ctx,
		config.HostAddress,
		config.APIPort,
		controller,
		publishers,
	)

	node := &Node{
		CleanupManager: config.CleanupManager,
		APIServer:      apiServer,
		IPFSClient:     config.IPFSClient,
		Controller:     controller,
		Transport:      config.Transport,
		ComputeNode:    computeNode,
		RequestorNode:  requesterNode,
		Executors:      executors,
		HostID:         config.HostID,
		metricsPort:    config.MetricsPort,
	}

	return node, nil
}
