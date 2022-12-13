package node

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute"
	"github.com/filecoin-project/bacalhau/pkg/eventhandler"
	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/bprotocol"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
)

type Requester struct {
	// Visible for testing
	Endpoint     requester.Endpoint
	nodeID       string
	JobStore     localdb.LocalDB
	computeProxy *bprotocol.ComputeProxy
	scheduler    *requester.Scheduler
}

//nolint:funlen
func NewRequesterNode(
	ctx context.Context,
	cm *system.CleanupManager,
	host host.Host,
	config RequesterConfig,
	jobStore localdb.LocalDB,
	verifiers verifier.VerifierProvider,
	storageProviders storage.StorageProvider,
	eventConsumer eventhandler.JobEventHandler) (*Requester, error) {
	computeProxy := bprotocol.NewComputeProxy(bprotocol.ComputeProxyParams{
		Host: host,
	})
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

	// register a handler for the bacalhau protocol handler that will forward requests to the scheduler
	bprotocol.NewCallbackHandler(bprotocol.CallbackHandlerParams{
		Host:     host,
		Callback: scheduler,
	})

	return &Requester{
		nodeID:       host.ID().String(),
		Endpoint:     endpoint,
		JobStore:     jobStore,
		scheduler:    scheduler,
		computeProxy: computeProxy,
	}, nil
}

func (r *Requester) RegisterLocalComputeEndpoint(endpoint compute.Endpoint) {
	r.computeProxy.RegisterLocalEndpoint(endpoint)
}
