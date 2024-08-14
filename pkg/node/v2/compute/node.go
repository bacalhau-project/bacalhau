package compute

import (
	"context"
	"errors"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

// TODO we need to expose this back to the caller as an interface
// this will allow a node to be created with a mocked out compute node and real requester.
// will also prevent callers from modifying an active compute node.

type Node struct {
	Name      string
	Transport *nats_transport.NATSTransport
	Server    *publicapi.Server
	Repo      *repo.FsRepo
	Config    v2.Compute

	EngineProvider    executor.ExecutorProvider
	PublisherProvider publisher.PublisherProvider
	StorageProvider   storage.StorageProvider
	CapacityProvider  CapacityProvider
	ExecutorService   ExecutorProvider
	EndpointProvider  EndpointProvider
	ManagementClient  *compute.ManagementClient
	LabelsProvider    models.LabelsProvider
	NodeInfoDecorator models.NodeInfoDecorator
}

func SetupNode(
	ctx context.Context,
	fsr *repo.FsRepo,
	server *publicapi.Server,
	transport *nats_transport.NATSTransport,
	name string,
	cfg v2.Compute,
) (*Node, error) {

	// .bacalhau/compute_store
	computePath, err := fsr.ComputePath()
	if err != nil {
		return nil, fmt.Errorf("opening compute storage path: %w", err)
	}
	// .bacalhau/compute_store/publishers
	publisherPath, err := fsr.PublisherPath()
	if err != nil {
		return nil, fmt.Errorf("opening executor storage path: %w", err)
	}

	engineProvider, err := NewEngineProvider(name, cfg.Executors)
	if err != nil {
		return nil, fmt.Errorf("creating executor provider: %w", err)
	}

	storageProvider, err := NewStorageProvider(cfg.InputSources)
	if err != nil {
		return nil, fmt.Errorf("creating storage provider: %w", err)
	}

	capacityProvider, err := NewCapacityProvider(ctx, computePath, cfg.Capacity, storageProvider)
	if err != nil {
		return nil, fmt.Errorf("creating capacity provider: %w", err)
	}

	publisherProvider, err := NewPublisherProvider(publisherPath, cfg.Publishers)
	if err != nil {
		return nil, fmt.Errorf("creating publisher provider: %w", err)
	}

	executorProvider, err := NewExecutorProvider(
		ctx,
		name,
		fsr,
		transport,
		engineProvider,
		storageProvider,
		publisherProvider,
		capacityProvider,
	)
	if err != nil {
		return nil, fmt.Errorf("creating executor provider: %w", err)
	}

	endpointProvider, err := NewComputeEndpointProvider(
		name,
		transport,
		server,
		cfg.Policy,
		storageProvider,
		engineProvider,
		publisherProvider,
		executorProvider,
		capacityProvider,
	)
	if err != nil {
		return nil, fmt.Errorf("creating endpoint provider: %w", err)
	}

	labelsProvider := models.MergeLabelsInOrder(
		&node.ConfigLabelsProvider{StaticLabels: cfg.Labels},
		&node.RuntimeLabelsProvider{},
		capacity.NewGPULabelsProvider(capacityProvider.Capacity()),
	)

	nodeInfoDecorator := compute.NewNodeInfoDecorator(compute.NodeInfoDecoratorParams{
		Executors:              engineProvider,
		Publisher:              publisherProvider,
		Storages:               storageProvider,
		MaxJobRequirements:     capacityProvider.Capacity(),
		RunningCapacityTracker: capacityProvider.RunningTracker(),
		QueueCapacityTracker:   capacityProvider.QueuedTracker(),
		ExecutorBuffer:         executorProvider.Executor(),
	})

	managementClient, err := SetupNetworkClient(
		name,
		fsr,
		cfg.Heartbeat,
		transport,
		capacityProvider,
		labelsProvider,
		nodeInfoDecorator,
	)
	if err != nil {
		return nil, fmt.Errorf("creating management client: %w", err)
	}

	return NewNode(
		name,
		cfg,
		fsr,
		server,
		transport,
		WithEngineProvider(engineProvider),
		WithPublisherProvider(publisherProvider),
		WithStorageProvider(storageProvider),
		WithCapacityProvider(capacityProvider),
		WithExecutorProvider(executorProvider),
		WithEndpointProvider(endpointProvider),
		WithManagementClient(managementClient),
		WithLabelsProvider(labelsProvider),
		WithNodeInfoDecorator(nodeInfoDecorator),
	)
}

func NewNode(
	name string,
	cfg v2.Compute,
	fsr *repo.FsRepo,
	server *publicapi.Server,
	transport *nats_transport.NATSTransport,
	opts ...Option,
) (*Node, error) {
	computeNode := &Node{
		Name:      name,
		Transport: transport,
		Server:    server,
		Repo:      fsr,
		Config:    cfg,
	}
	// Apply options
	for _, opt := range opts {
		if err := opt(computeNode); err != nil {
			return nil, err
		}
	}

	// Validate and return
	if err := computeNode.validate(); err != nil {
		return nil, err
	}
	return computeNode, nil
}

func (n *Node) Start(ctx context.Context) error {
	if err := n.ExecutorService.Start(ctx); err != nil {
		return fmt.Errorf("starting executor service: %w", err)
	}
	if err := n.ManagementClient.RegisterNode(ctx); err != nil {
		return fmt.Errorf("registering compute node with orchestrator: %s", err)
	}
	go n.ManagementClient.Start(ctx)

	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	var stopErr error
	n.ManagementClient.Stop()
	if err := n.ExecutorService.Stop(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("stopping executor service: %w", err))
	}

	return stopErr
}

func (n *Node) validate() error {
	return nil
}
