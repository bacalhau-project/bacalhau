package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/rs/zerolog/log"
)

const JOB_EVENT_CHANNEL = "bacalhau-job-event"

type Libp2pTransport struct {
	Ctx context.Context

	// the writer we emit events through
	GenericTransport *transport.GenericTransport

	Host                 host.Host
	Port                 int
	PubSub               *pubsub.PubSub
	JobEventTopic        *pubsub.Topic
	JobEventSubscription *pubsub.Subscription
}

func makeLibp2pHost(
	port int,
) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	// TODO: allow the user to provide an existing keypair
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Error().Err(err)
		return nil, err
	}

	// 0.0.0.0 will listen on any interface device.
	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))

	// libp2p.New constructs a new libp2p Host.
	// Other options can be added here.
	return libp2p.New(
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
}

func NewLibp2pTransport(
	ctx context.Context,
	port int,
) (*Libp2pTransport, error) {
	host, err := makeLibp2pHost(port)
	if err != nil {
		return nil, err
	}
	pubsub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}
	jobEventTopic, err := pubsub.Join(JOB_EVENT_CHANNEL)
	if err != nil {
		return nil, err
	}
	jobEventSubscription, err := jobEventTopic.Subscribe()
	if err != nil {
		return nil, err
	}

	libp2pTransport := &Libp2pTransport{
		Ctx:                  ctx,
		Host:                 host,
		Port:                 port,
		PubSub:               pubsub,
		JobEventTopic:        jobEventTopic,
		JobEventSubscription: jobEventSubscription,
	}

	// setup the event writer
	libp2pTransport.GenericTransport = transport.NewGenericTransport(
		host.ID().String(),
		func(event *types.JobEvent) error {
			return libp2pTransport.writeJobEvent(event)
		},
	)

	return libp2pTransport, nil
}

/*

  PUBLIC INTERFACE

*/

func (scheduler *Libp2pTransport) HostId() (string, error) {
	return scheduler.Host.ID().String(), nil
}

func (scheduler *Libp2pTransport) Start() error {
	if len(scheduler.GenericTransport.SubscribeFuncs) <= 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}
	go scheduler.readLoopJobEvents()
	go func() {
		log.Debug().Msg("Libp2p scheduler has started")
		<-scheduler.Ctx.Done()
		scheduler.Host.Close()
		log.Debug().Msg("Libp2p scheduler has stopped")
	}()
	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pTransport) List() (types.ListResponse, error) {
	return scheduler.GenericTransport.List()
}

func (scheduler *Libp2pTransport) Get(id string) (*types.Job, error) {
	return scheduler.GenericTransport.Get(id)
}

func (scheduler *Libp2pTransport) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	scheduler.GenericTransport.Subscribe(subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pTransport) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	return scheduler.GenericTransport.SubmitJob(spec, deal)
}

func (scheduler *Libp2pTransport) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return scheduler.GenericTransport.UpdateDeal(jobId, deal)
}

func (scheduler *Libp2pTransport) CancelJob(jobId string) error {
	return nil
}

func (scheduler *Libp2pTransport) AcceptJobBid(jobId, nodeId string) error {
	return scheduler.GenericTransport.AcceptJobBid(jobId, nodeId)
}

func (scheduler *Libp2pTransport) RejectJobBid(jobId, nodeId, message string) error {
	return scheduler.GenericTransport.RejectJobBid(jobId, nodeId, message)
}

func (scheduler *Libp2pTransport) AcceptResult(jobId, nodeId string) error {
	return scheduler.GenericTransport.AcceptResult(jobId, nodeId)
}

func (scheduler *Libp2pTransport) RejectResult(jobId, nodeId, message string) error {
	return scheduler.GenericTransport.RejectResult(jobId, nodeId, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pTransport) BidJob(jobId string) error {
	return scheduler.GenericTransport.BidJob(jobId)
}

func (scheduler *Libp2pTransport) SubmitResult(jobId, status, resultsId string) error {
	return scheduler.GenericTransport.SubmitResult(jobId, status, resultsId)
}

func (scheduler *Libp2pTransport) ErrorJob(jobId, status string) error {
	return scheduler.GenericTransport.ErrorJob(jobId, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (scheduler *Libp2pTransport) ErrorJobForNode(jobId, nodeId, status string) error {
	return scheduler.GenericTransport.ErrorJobForNode(jobId, nodeId, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pTransport) Connect(peerConnect string) error {

	if peerConnect == "" {
		return nil
	}
	maddr, err := multiaddr.NewMultiaddr(peerConnect)
	if err != nil {
		return err
	}

	// Extract the peer ID from the multiaddr.
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return err
	}

	scheduler.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	//nolint
	scheduler.Host.Connect(scheduler.Ctx, *info)

	return nil
}

func (scheduler *Libp2pTransport) writeJobEvent(event *types.JobEvent) error {
	msgBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Sending event: %s", string(msgBytes))
	return scheduler.JobEventTopic.Publish(scheduler.Ctx, msgBytes)
}

func (scheduler *Libp2pTransport) readLoopJobEvents() {
	for {
		msg, err := scheduler.JobEventSubscription.Next(scheduler.Ctx)
		if err != nil {
			return
		}

		jobEvent := new(types.JobEvent)
		err = json.Unmarshal(msg.Data, jobEvent)
		if err != nil {
			continue
		}

		scheduler.GenericTransport.ReadEvent(jobEvent)
	}
}
