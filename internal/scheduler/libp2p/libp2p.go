package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

type Libp2pScheduler struct {
	Ctx context.Context

	// the jobs we have already filtered and might want to process
	//Jobs []types.Job

	// see types.Update for the message that updates these fields

	// a map of job id into bacalhau nodes that are in progress of doing some work with their claimed state and human readable statuses
	JobState  map[string]map[string]string
	JobStatus map[string]map[string]string
	// a map of job id onto bacalhau compute nodes that claim to have done the work onto cids of the job results published by them
	JobResults map[string]map[string]string

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
		Ctx: ctx,
		//Jobs:                  []types.Job{},
		Host:                  host,
		PubSub:                pubsub,
		JobState:              make(map[string]map[string]string),
		JobStatus:             make(map[string]map[string]string),
		JobResults:            make(map[string]map[string]string),
		JobCreateTopic:        jobCreateTopic,
		JobCreateSubscription: jobCreateSubscription,
		JobUpdateTopic:        jobUpdateTopic,
		JobUpdateSubscription: jobUpdateSubscription,
	}
	go scheduler.ReadLoopJobCreate()
	go scheduler.ReadLoopJobUpdate()
	go func() {
		fmt.Printf("waiting for bacalhau libp2p context done\n")
		<-ctx.Done()
		fmt.Printf("closing bacalhau libp2p daemon\n")
		host.Close()
		fmt.Printf("closed bacalhau libp2p daemon\n")
	}()
	return scheduler, nil
}

func (scheduler *Libp2pScheduler) ReadLoopJobCreate() {
	for {
		msg, err := scheduler.JobCreateSubscription.Next(scheduler.Ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == scheduler.Host.ID() {
			continue
		}
		// job := new(types.Job)
		// err = json.Unmarshal(msg.Data, job)
		// if err != nil {
		// 	continue
		// }
		//go scheduler.AddJob(job)
	}
}

func (scheduler *Libp2pScheduler) ReadLoopJobUpdate() {
	for {
		msg, err := scheduler.JobUpdateSubscription.Next(scheduler.Ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == scheduler.Host.ID() {
			continue
		}
		jobUpdate := new(types.JobState)
		err = json.Unmarshal(msg.Data, jobUpdate)
		if err != nil {
			continue
		}
		//go scheduler.UpdateJob(jobUpdate)
	}
}
