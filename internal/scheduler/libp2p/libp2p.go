package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

type Libp2pScheduler struct {
	Ctx context.Context

	// the jobs we have already filtered and might want to process
	Jobs map[string]*types.JobData

	// the list of functions to call when we get an update about a job
	SubscribeFuncs []func(eventName string, job *types.JobData)

	Host                  host.Host
	PubSub                *pubsub.PubSub
	JobCreateTopic        *pubsub.Topic
	JobCreateSubscription *pubsub.Subscription
	JobUpdateTopic        *pubsub.Topic
	JobUpdateSubscription *pubsub.Subscription
}

func makeLibp2pHost(
	port int,
) (host.Host, error) {
	// Creates a new RSA key pair for this host.
	// TODO: allow the user to provide an existing keypair
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
	if err != nil {
		log.Println(err)
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

func NewLibp2pScheduler(
	ctx context.Context,
	port int,
) (*Libp2pScheduler, error) {
	host, err := makeLibp2pHost(port)
	if err != nil {
		return nil, err
	}
	pubsub, err := pubsub.NewGossipSub(ctx, host)
	if err != nil {
		return nil, err
	}
	jobCreateTopic, err := pubsub.Join("bacalhau-jobs-create")
	if err != nil {
		return nil, err
	}
	jobCreateSubscription, err := jobCreateTopic.Subscribe()
	if err != nil {
		return nil, err
	}
	jobUpdateTopic, err := pubsub.Join("bacalhau-jobs-update")
	if err != nil {
		return nil, err
	}
	jobUpdateSubscription, err := jobUpdateTopic.Subscribe()
	if err != nil {
		return nil, err
	}
	scheduler := &Libp2pScheduler{
		Ctx:                   ctx,
		Host:                  host,
		PubSub:                pubsub,
		Jobs:                  make(map[string]*types.JobData),
		JobCreateTopic:        jobCreateTopic,
		JobCreateSubscription: jobCreateSubscription,
		JobUpdateTopic:        jobUpdateTopic,
		JobUpdateSubscription: jobUpdateSubscription,
	}
	return scheduler, nil
}

/*

  PUBLIC INTERFACE

*/

func (scheduler *Libp2pScheduler) HostId() (string, error) {
	return scheduler.Host.ID().String(), nil
}

func (scheduler *Libp2pScheduler) Start() error {
	if len(scheduler.SubscribeFuncs) <= 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}
	go scheduler.ReadLoopJobCreate()
	go scheduler.ReadLoopJobUpdate()
	go func() {
		fmt.Printf("waiting for bacalhau libp2p context done\n")
		<-scheduler.Ctx.Done()
		fmt.Printf("closing bacalhau libp2p daemon\n")
		scheduler.Host.Close()
		fmt.Printf("closed bacalhau libp2p daemon\n")
	}()
	return nil
}

func (scheduler *Libp2pScheduler) List() (types.ListResponse, error) {
	return types.ListResponse{
		Jobs: scheduler.Jobs,
	}, nil
}

func (scheduler *Libp2pScheduler) Subscribe(f func(eventName string, job *types.JobData)) {
	scheduler.SubscribeFuncs = append(scheduler.SubscribeFuncs, f)
}

func (scheduler *Libp2pScheduler) SubmitJob(spec *types.JobSpec) error {
	msgBytes, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	return scheduler.JobCreateTopic.Publish(scheduler.Ctx, msgBytes)
}

func (scheduler *Libp2pScheduler) CancelJob(jobId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) ApproveJobBid(jobId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) RejectJobBid(jobId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) UpdateJob(jobId, field, value string) error {
	return nil
}

func (scheduler *Libp2pScheduler) UpdateJobState(jobId string, update *types.JobState) error {
	nodeId, err := scheduler.HostId()
	if err != nil {
		return err
	}
	update.JobId = jobId
	update.NodeId = nodeId
	msgBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	return scheduler.JobUpdateTopic.Publish(scheduler.Ctx, msgBytes)
}

func (scheduler *Libp2pScheduler) ApproveResult(jobId, resultId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) RejectResult(jobId, resultId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) BidJob(jobId string) (string, error) {
	scheduler.UpdateJobState(jobId, &types.JobState{
		State: system.JOB_STATE_BIDDING,
	})
	return "", nil
}

func (scheduler *Libp2pScheduler) SubmitProgress(jobId, resultId, state, status string, resultPointer *string) error {
	return nil
}

/*

  INTERNAL IMPLEMENTATION

*/

func (scheduler *Libp2pScheduler) Connect(peerConnect string) error {

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

func (scheduler *Libp2pScheduler) TriggerJobEvent(id, eventName string) {
	for _, subscribeFunc := range scheduler.SubscribeFuncs {
		subscribeFunc(eventName, scheduler.Jobs[id])
	}
}

func (scheduler *Libp2pScheduler) ReadLoopJobCreate() {
	for {
		msg, err := scheduler.JobCreateSubscription.Next(scheduler.Ctx)
		if err != nil {
			return
		}

		job := new(types.JobSpec)
		err = json.Unmarshal(msg.Data, job)
		if err != nil {
			continue
		}

		scheduler.Jobs[job.Id] = &types.JobData{
			Job:   job,
			State: make(map[string]*types.JobState),
		}

		scheduler.TriggerJobEvent(job.Id, system.JOB_EVENT_CREATED)
	}
}

func (scheduler *Libp2pScheduler) ReadLoopJobUpdate() {
	for {
		msg, err := scheduler.JobUpdateSubscription.Next(scheduler.Ctx)
		if err != nil {
			return
		}

		jobUpdate := new(types.JobState)
		err = json.Unmarshal(msg.Data, jobUpdate)
		if err != nil {
			continue
		}

		scheduler.Jobs[jobUpdate.JobId].State[jobUpdate.NodeId] = jobUpdate

		scheduler.TriggerJobEvent(jobUpdate.JobId, system.JOB_EVENT_UPDATED)
	}
}
