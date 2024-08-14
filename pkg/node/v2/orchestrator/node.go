package orchestrator

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	v2 "github.com/bacalhau-project/bacalhau/pkg/config/types/v2"
	"github.com/bacalhau-project/bacalhau/pkg/eventhandler"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	boltjobstore "github.com/bacalhau-project/bacalhau/pkg/jobstore/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	nats_transport "github.com/bacalhau-project/bacalhau/pkg/nats/transport"
	"github.com/bacalhau-project/bacalhau/pkg/node/manager"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/selection/discovery"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	auth_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/auth"
	orchestrator_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/orchestrator"
	requester_endpoint "github.com/bacalhau-project/bacalhau/pkg/publicapi/endpoint/requester"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type Node struct {
	Name                  string
	Transport             *nats_transport.NATSTransport
	Server                *publicapi.Server
	Repo                  *repo.FsRepo
	Config                v2.Orchestrator
	Scheduler             *SchedulerService
	Store                 jobstore.Store
	Endpoint              *requester.BaseEndpoint
	NodeManager           *manager.NodeManager
	NodeInfoStore         routing.NodeInfoStore
	TracerContextProvider *eventhandler.TracerContextProvider
	EventTracer           *eventhandler.Tracer
	DebugInfoProvider     []models.DebugInfoProvider
}

func (n *Node) Start(ctx context.Context) error {
	if err := n.Transport.RegisterNodeInfoConsumer(ctx, n.NodeInfoStore); err != nil {
		return fmt.Errorf("registering node info store on transport: %w", err)
	}
	if err := n.NodeManager.Start(ctx); err != nil {
		return fmt.Errorf("starting node manager: %w", err)
	}
	if err := n.Transport.RegisterManagementEndpoint(n.NodeManager); err != nil {
		return fmt.Errorf("registering node manager on transport: %w", err)
	}
	if err := n.Transport.RegisterComputeCallback(n.Endpoint); err != nil {
		return fmt.Errorf("registering endpoint on transport: %w", err)
	}
	if err := n.Scheduler.Start(ctx); err != nil {
		return fmt.Errorf("starting scheduler service: %w", err)
	}
	return nil
}

func (n *Node) Stop(ctx context.Context) error {
	var stopErr error
	if err := n.Scheduler.Stop(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("stopping scheduler service: %w", err))
	}
	if err := n.TracerContextProvider.Shutdown(); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("stopping tracer context provider: %w", err))
	}
	if err := n.EventTracer.Shutdown(); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("stopping event tracer: %w", err))
	}
	if err := n.Store.Close(ctx); err != nil {
		stopErr = errors.Join(stopErr, fmt.Errorf("closing job store: %w", err))
	}

	return stopErr
}

func (n *Node) validate() error {
	return nil
}

func SetupNode(
	ctx context.Context,
	name string,
	cfg v2.Orchestrator,
	fsr *repo.FsRepo,
	server *publicapi.Server,
	transport *nats_transport.NATSTransport,
	authProvider authn.Provider,
) (*Node, error) {
	// .bacalhau/orchestrator_store/jobs.db
	jobStorePath, err := fsr.JobStorePath()
	if err != nil {
		return nil, fmt.Errorf("opening job store: %w", err)
	}
	jobStore, err := boltjobstore.NewBoltJobStore(jobStorePath)
	if err != nil {
		return nil, fmt.Errorf("creating job store: %w", err)
	}

	nodeInfoStore, err := SetupNodeInfoStore(ctx, transport)
	if err != nil {
		return nil, err
	}

	nodeManager, err := SetupNodeManager(ctx, transport, nodeInfoStore, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating node manager: %w", err)
	}

	// prepare event handlers
	tracerContextProvider := eventhandler.NewTracerContextProvider(name)
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	eventEmitter := orchestrator.NewEventEmitter(orchestrator.EventEmitterParams{
		EventConsumer: localJobEventConsumer,
	})

	scheduler, err := SetupSchedulerService(
		name,
		cfg,
		transport.ComputeProxy(),
		jobStore,
		eventEmitter,
		nodeManager,
		// TODO this will need to be an option on the node
		retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true}),
	)
	if err != nil {
		return nil, fmt.Errorf("creating scheduler service: %w", err)
	}

	// result transformers that are applied to the result before it is returned to the user
	resultTransformers := transformer.ChainedTransformer[*models.SpecConfig]{}

	jobTransformers := transformer.ChainedTransformer[*models.Job]{
		transformer.JobFn(transformer.IDGenerator),
		transformer.NameOptional(),
		//transformer.DefaultsApplier(requesterConfig.JobDefaults),
		transformer.RequesterInfo(name),
	}

	endpointV2 := orchestrator.NewBaseEndpoint(&orchestrator.BaseEndpointParams{
		ID:                name,
		Store:             jobStore,
		EventEmitter:      eventEmitter,
		ComputeProxy:      transport.ComputeProxy(),
		JobTransformer:    jobTransformers,
		TaskTranslator:    nil,
		ResultTransformer: resultTransformers,
	})

	// TODO: delete this when we are ready to stop serving a deprecation notice.
	requester_endpoint.NewEndpoint(server.Router)

	orchestrator_endpoint.NewEndpoint(orchestrator_endpoint.EndpointParams{
		Router:       server.Router,
		Orchestrator: endpointV2,
		JobStore:     jobStore,
		NodeManager:  nodeManager,
	})

	auth_endpoint.BindEndpoint(ctx, server.Router, authProvider)

	// TODO is there any point is making this thing?
	eventTracer, err := eventhandler.NewTracer(os.DevNull)
	if err != nil {
		return nil, err
	}

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	localJobEventConsumer.AddHandlers(
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// record the event in a log
		eventTracer,
	)

	// This endpoint implements the protocol formerly known as `bprotocol`.
	// It provides the compute call back endpoints for interacting with compute nodes.
	// e.g. bidding, job completions, cancellations, and failures
	endpoint := requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:           name,
		EventEmitter: eventEmitter,
		Store:        jobStore,
	})

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []models.DebugInfoProvider{
		discovery.NewDebugInfoProvider(nodeManager),
	}

	return NewNode(
		name,
		cfg,
		fsr,
		server,
		transport,
		WithScheduler(scheduler),
		WithEndpoint(endpoint),
		WithTracerContextProvider(tracerContextProvider),
		WithDebugInfoProvider(debugInfoProviders...),
		WithNodeInfoStore(nodeInfoStore),
		WithNodeManer(nodeManager),
		WithJobStore(jobStore),
		WithEventTracer(eventTracer),
	)
}

func NewNode(
	name string,
	cfg v2.Orchestrator,
	fsr *repo.FsRepo,
	server *publicapi.Server,
	transport *nats_transport.NATSTransport,
	opts ...Option,
) (*Node, error) {
	orchestratorNode := &Node{
		Name:      name,
		Transport: transport,
		Server:    server,
		Repo:      fsr,
		Config:    cfg,
	}
	// Apply options
	for _, opt := range opts {
		if err := opt(orchestratorNode); err != nil {
			return nil, err
		}
	}

	// Validate and return
	if err := orchestratorNode.validate(); err != nil {
		return nil, err
	}
	return orchestratorNode, nil
}
