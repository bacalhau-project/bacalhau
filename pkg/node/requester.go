package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/requester/discovery"
	"github.com/filecoin-project/bacalhau/pkg/requester/nodestore"
	requester_publicapi "github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requester/ranking"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	simulator_protocol "github.com/filecoin-project/bacalhau/pkg/transport/simulator"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
)

type Requester struct {
	// Visible for testing
	Endpoint      requester.Endpoint
	JobStore      localdb.LocalDB
	computeProxy  *bprotocol.ComputeProxy
	localCallback *requester.Scheduler
	cleanupFunc   func(ctx context.Context)
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	cleanupManager *system.CleanupManager,
	host host.Host,
	apiServer *publicapi.APIServer,
	config RequesterConfig,
	jobStore localdb.LocalDB,
	simulatorNodeID string,
	simulatorRequestHandler *simulator.RequestHandler,
	verifiers verifier.VerifierProvider,
	storageProviders storage.StorageProvider,
	nodeInfoPubSub pubsub.PubSub[model.NodeInfo],
	gossipSub *libp2p_pubsub.PubSub,
) (*Requester, error) {
	// prepare event handlers
	tracerContextProvider := system.NewTracerContextProvider(host.ID().String())
	localJobEventConsumer := eventhandler.NewChainedJobEventHandler(tracerContextProvider)

	// compute proxy
	var computeProxy compute.Endpoint
	standardComputeProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host: host,
	})
	// if we are running in simulator mode, then we use the simulator proxy to forward all requests to th simulator node.
	if simulatorNodeID != "" {
		simulatorProxy := simulator_protocol.NewComputeProxy(simulator_protocol.ComputeProxyParams{
			SimulatorNodeID: simulatorNodeID,
			Host:            host,
		})
		if simulatorRequestHandler != nil {
			// if this node is the simulator node, we need to register a local endpoint to allow self dialing
			simulatorProxy.RegisterLocalComputeEndpoint(simulatorRequestHandler)
			// set standard endpoint implementation so that the simulator can forward requests to the correct endpoints
			// after it finishes its validation and processing of the request
			simulatorRequestHandler.SetComputeProxy(standardComputeProxy)
		}
		computeProxy = simulatorProxy
	} else {
		computeProxy = standardComputeProxy
	}

	// compute node discoverer
	nodeInfoStore := nodestore.NewInMemoryNodeInfoStore(nodestore.InMemoryNodeInfoStoreParams{
		TTL: config.NodeInfoStoreTTL,
	})
	nodeDiscoverer := discovery.NewChained(true)
	nodeDiscoverer.Add(
		discovery.NewStoreNodeDiscoverer(discovery.StoreNodeDiscovererParams{
			Host:         host,
			Store:        nodeInfoStore,
			PeerStoreTTL: config.DiscoveredPeerStoreTTL,
		}),
		discovery.NewIdentityNodeDiscoverer(discovery.IdentityNodeDiscovererParams{
			Host: host,
		}),
	)

	scheduler := requester.NewScheduler(ctx, cleanupManager, requester.SchedulerParams{
		ID:             host.ID().String(),
		JobStore:       jobStore,
		NodeDiscoverer: nodeDiscoverer,
		NodeRanker: ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: config.NodeRankRandomnessRange,
		}),
		ComputeEndpoint:  computeProxy,
		Verifiers:        verifiers,
		StorageProviders: storageProviders,
		EventEmitter: requester.NewEventEmitter(requester.EventEmitterParams{
			EventConsumer: localJobEventConsumer,
		}),
		JobNegotiationTimeout:              config.JobNegotiationTimeout,
		StateManagerBackgroundTaskInterval: config.StateManagerBackgroundTaskInterval,
	})

	publicKey := host.Peerstore().PubKey(host.ID())
	marshaledPublicKey, err := crypto.MarshalPublicKey(publicKey)

	if err != nil {
		return nil, err
	}
	endpoint := requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         host.ID().String(),
		PublicKey:                  marshaledPublicKey,
		JobStore:                   jobStore,
		Scheduler:                  scheduler,
		Verifiers:                  verifiers,
		StorageProviders:           storageProviders,
		MinJobExecutionTimeout:     config.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
	})

	// if this node is the simulator, then we pass incoming requests to the simulator before passing them to the endpoint
	if simulatorRequestHandler != nil {
		bprotocol.NewCallbackHandler(bprotocol.CallbackHandlerParams{
			Host:     host,
			Callback: simulatorRequestHandler,
		})
	} else {
		// register a handler for the bacalhau protocol handler that will forward requests to the scheduler
		bprotocol.NewCallbackHandler(bprotocol.CallbackHandlerParams{
			Host:     host,
			Callback: scheduler,
		})
	}

	// register debug info providers for the /debug endpoint
	debugInfoProviders := []model.DebugInfoProvider{
		requester.NewScheduledJobsInfoProvider(requester.ScheduledJobsInfoProviderParams{
			Name:      "ScheduledJobs",
			Scheduler: scheduler,
		}),
	}

	// register requester public http apis
	requesterAPIServer := requester_publicapi.NewRequesterAPIServer(requester_publicapi.RequesterAPIServerParams{
		APIServer:          apiServer,
		Requester:          endpoint,
		DebugInfoProviders: debugInfoProviders,
		LocalDB:            jobStore,
		StorageProviders:   storageProviders,
	})
	err = requesterAPIServer.RegisterAllHandlers()
	if err != nil {
		return nil, err
	}

	// PubSub to publish job events to the network
	libp2p2JobEventPubSub, err := libp2p.NewPubSub[pubsub.BufferingEnvelope](libp2p.PubSubParams{
		Host:        host,
		TopicName:   JobEventsTopic,
		PubSub:      gossipSub,
		IgnoreLocal: true,
	})
	if err != nil {
		return nil, err
	}

	bufferedJobEventPubSub := pubsub.NewBufferingPubSub[model.JobEvent](pubsub.BufferingPubSubParams{
		DelegatePubSub: libp2p2JobEventPubSub,
		MaxBufferSize:  32 * 1024,       //nolint:gomnd
		MaxBufferAge:   5 * time.Second, // increase this once we move to an external job storage
	})

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(host.ID().String())
	localDBEventHandler := localdb.NewLocalDBEventHandler(jobStore)
	eventTracer, err := eventhandler.NewTracer()
	if err != nil {
		return nil, err
	}

	// order of event handlers is important as triggering some handlers might depend on the state of others.
	localJobEventConsumer.AddHandlers(
		// add tracing metadata to the context about the read event
		eventhandler.JobEventHandlerFunc(lifecycleEventHandler.HandleConsumedJobEvent),
		// ends the span for the job if received a terminal event
		tracerContextProvider,
		// record the event in a log
		eventTracer,
		// update the job state in the local DB
		localDBEventHandler,
		// dispatches events to listening websockets
		requesterAPIServer,
		// dispatches events to the network
		eventhandler.JobEventHandlerFunc(bufferedJobEventPubSub.Publish),
	)

	// register consumers of job events publishes over gossipSub
	networkJobEventConsumer := eventhandler.NewChainedJobEventHandler(system.NewNoopContextProvider())
	networkJobEventConsumer.AddHandlers(
		// update the job state in the local DB
		localDBEventHandler,
	)
	err = bufferedJobEventPubSub.Subscribe(ctx, pubsub.SubscriberFunc[model.JobEvent](networkJobEventConsumer.HandleJobEvent))
	if err != nil {
		return nil, err
	}

	// register consumers of node info published over gossipSub
	nodeInfoSubscriber := pubsub.NewChainedSubscriber[model.NodeInfo](true)
	nodeInfoSubscriber.Add(pubsub.SubscriberFunc[model.NodeInfo](nodeInfoStore.Add))
	err = nodeInfoPubSub.Subscribe(ctx, nodeInfoSubscriber)
	if err != nil {
		return nil, err
	}

	// A single cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		cleanupErr := bufferedJobEventPubSub.Close(ctx)
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close job event pubsub")
		}
		cleanupErr = libp2p2JobEventPubSub.Close(ctx)
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to close libp2p job event pubsub")
		}

		cleanupErr = tracerContextProvider.Shutdown()
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to shutdown tracer context provider")
		}

		cleanupErr = eventTracer.Shutdown()
		if cleanupErr != nil {
			log.Error().Err(cleanupErr).Msg("failed to shutdown event tracer")
		}
	}

	return &Requester{
		Endpoint:      endpoint,
		localCallback: scheduler,
		JobStore:      jobStore,
		computeProxy:  standardComputeProxy,
		cleanupFunc:   cleanupFunc,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalComputeEndpoint(endpoint)
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
