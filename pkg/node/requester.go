package node

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/jobstore"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/filecoin-project/bacalhau/pkg/pubsub/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/requester/discovery"
	requester_publicapi "github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requester/ranking"
	"github.com/filecoin-project/bacalhau/pkg/routing"
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
	Endpoint           requester.Endpoint
	JobStore           jobstore.Store
	computeProxy       *bprotocol.ComputeProxy
	localCallback      *requester.Scheduler
	requesterAPIServer *requester_publicapi.RequesterAPIServer
	cleanupFunc        func(ctx context.Context)
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	cleanupManager *system.CleanupManager,
	host host.Host,
	apiServer *publicapi.APIServer,
	config RequesterConfig,
	jobStore jobstore.Store,
	simulatorNodeID string,
	simulatorRequestHandler *simulator.RequestHandler,
	verifiers verifier.VerifierProvider,
	storageProviders storage.StorageProvider,
	gossipSub *libp2p_pubsub.PubSub,
	nodeInfoStore routing.NodeInfoStore,
) (*Requester, error) {
	// prepare event handlers
	tracerContextProvider := eventhandler.NewTracerContextProvider(host.ID().String())
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
	nodeDiscoveryChain := discovery.NewChain(true)
	nodeDiscoveryChain.Add(
		discovery.NewStoreNodeDiscoverer(discovery.StoreNodeDiscovererParams{
			Store: nodeInfoStore,
		}),
		discovery.NewIdentityNodeDiscoverer(discovery.IdentityNodeDiscovererParams{
			Host: host,
		}),
	)

	// compute node ranker
	nodeRankerChain := ranking.NewChain()
	nodeRankerChain.Add(
		// rankers that act as filters and give a -1 score to nodes that do not match the filter
		ranking.NewEnginesNodeRanker(),
		ranking.NewLabelsNodeRanker(),
		ranking.NewMaxUsageNodeRanker(),

		// arbitrary rankers
		ranking.NewRandomNodeRanker(ranking.RandomNodeRankerParams{
			RandomnessRange: config.NodeRankRandomnessRange,
		}),
	)

	scheduler := requester.NewScheduler(requester.SchedulerParams{
		ID:               host.ID().String(),
		Host:             host,
		JobStore:         jobStore,
		NodeDiscoverer:   nodeDiscoveryChain,
		NodeRanker:       nodeRankerChain,
		ComputeEndpoint:  computeProxy,
		Verifiers:        verifiers,
		StorageProviders: storageProviders,
		EventEmitter: requester.NewEventEmitter(requester.EventEmitterParams{
			EventConsumer: localJobEventConsumer,
		}),
	})

	publicKey := host.Peerstore().PubKey(host.ID())
	marshaledPublicKey, err := crypto.MarshalPublicKey(publicKey)

	if err != nil {
		return nil, err
	}
	endpoint := requester.NewBaseEndpoint(&requester.BaseEndpointParams{
		ID:                         host.ID().String(),
		PublicKey:                  marshaledPublicKey,
		Scheduler:                  scheduler,
		Verifiers:                  verifiers,
		StorageProviders:           storageProviders,
		MinJobExecutionTimeout:     config.MinJobExecutionTimeout,
		DefaultJobExecutionTimeout: config.DefaultJobExecutionTimeout,
	})

	housekeeping := requester.NewHousekeeping(requester.HousekeepingParams{
		Endpoint: endpoint,
		JobStore: jobStore,
		NodeID:   host.ID().String(),
		Interval: config.HousekeepingBackgroundTaskInterval,
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
	debugInfoProviders := []model.DebugInfoProvider{}

	// register requester public http apis
	requesterAPIServer := requester_publicapi.NewRequesterAPIServer(requester_publicapi.RequesterAPIServerParams{
		APIServer:          apiServer,
		Requester:          endpoint,
		DebugInfoProviders: debugInfoProviders,
		JobStore:           jobStore,
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
		MaxBufferSize:  1, //nolint:gomnd // increase this once we move to an external job storage
		MaxBufferAge:   1 * time.Minute,
	})

	// Register event handlers
	lifecycleEventHandler := system.NewJobLifecycleEventHandler(host.ID().String())
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
		// dispatches events to listening websockets
		requesterAPIServer,
		// dispatches events to the network
		eventhandler.JobEventHandlerFunc(bufferedJobEventPubSub.Publish),
	)

	// A single cleanup function to make sure the order of closing dependencies is correct
	cleanupFunc := func(ctx context.Context) {
		// stop the housekeeping background task
		housekeeping.Stop()

		cleanupErr := bufferedJobEventPubSub.Close(ctx)
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to close job event pubsub")
		}
		cleanupErr = libp2p2JobEventPubSub.Close(ctx)
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to close libp2p job event pubsub")
		}

		cleanupErr = tracerContextProvider.Shutdown()
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to shutdown tracer context provider")
		}

		cleanupErr = eventTracer.Shutdown()
		if cleanupErr != nil {
			log.Ctx(ctx).Error().Err(cleanupErr).Msg("failed to shutdown event tracer")
		}
	}

	return &Requester{
		Endpoint:           endpoint,
		localCallback:      scheduler,
		JobStore:           jobStore,
		computeProxy:       standardComputeProxy,
		cleanupFunc:        cleanupFunc,
		requesterAPIServer: requesterAPIServer,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalComputeEndpoint(endpoint)
}

func (r *Requester) cleanup(ctx context.Context) {
	r.cleanupFunc(ctx)
}
