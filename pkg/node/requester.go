package node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/simulator"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	simulator_protocol "github.com/filecoin-project/bacalhau/pkg/transport/simulator"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

type Requester struct {
	// Visible for testing
	Endpoint      requester.Endpoint
	host          host.Host
	JobStore      localdb.LocalDB
	computeProxy  *bprotocol.ComputeProxy
	localCallback *requester.Scheduler
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	cm *system.CleanupManager,
	host host.Host,
	config RequesterConfig,
	jobStore localdb.LocalDB,
	simulatorNodeID string,
	simulatorRequestHandler *simulator.RequestHandler,
	verifiers verifier.VerifierProvider,
	storageProviders storage.StorageProvider,
	eventConsumer eventhandler.JobEventHandler) (*Requester, error) {
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

	scheduler := requester.NewScheduler(ctx, cm, requester.SchedulerParams{
		ID:       host.ID().String(),
		JobStore: jobStore,
		NodeDiscoverer: requester.NewIdentityNodeDiscoverer(requester.IdentityNodeDiscovererParams{
			Host: host,
		}),
		NodeRanker: requester.NewRandomNodeRanker(requester.RandomNodeRankerParams{
			RandomnessRange: config.NodeRankRandomnessRange,
		}),
		ComputeEndpoint:  computeProxy,
		Verifiers:        verifiers,
		StorageProviders: storageProviders,
		EventEmitter: requester.NewEventEmitter(requester.EventEmitterParams{
			EventConsumer: eventConsumer,
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

	return &Requester{
		host:          host,
		Endpoint:      endpoint,
		JobStore:      jobStore,
		localCallback: scheduler,
		computeProxy:  standardComputeProxy,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalComputeEndpoint(endpoint)
}
