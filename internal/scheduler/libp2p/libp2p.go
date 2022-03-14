package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/bacalhau/internal/system"
	"github.com/filecoin-project/bacalhau/internal/types"
	"github.com/google/uuid"
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

type Libp2pScheduler struct {
	Ctx context.Context

	Jobs map[string]*types.Job

	// the list of functions to call when we get an update about a job
	SubscribeFuncs []func(jobEvent *types.JobEvent, job *types.Job)

	Host                 host.Host
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
	jobEventTopic, err := pubsub.Join(JOB_EVENT_CHANNEL)
	if err != nil {
		return nil, err
	}
	jobEventSubscription, err := jobEventTopic.Subscribe()
	if err != nil {
		return nil, err
	}
	scheduler := &Libp2pScheduler{
		Ctx:                  ctx,
		Host:                 host,
		PubSub:               pubsub,
		Jobs:                 make(map[string]*types.Job),
		JobEventTopic:        jobEventTopic,
		JobEventSubscription: jobEventSubscription,
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
	go scheduler.readLoopJobEvents()
	go func() {
		log.Debug().Msg("Waiting for bacalhau libp2p context to finish.\n")
		<-scheduler.Ctx.Done()
		log.Debug().Msg("Closing bacalhau libp2p daemon\n")
		scheduler.Host.Close()
		log.Debug().Msg("Closed bacalhau libp2p daemon\n")
	}()
	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pScheduler) List() (types.ListResponse, error) {
	return types.ListResponse{
		Jobs: scheduler.Jobs,
	}, nil
}

func (scheduler *Libp2pScheduler) Get(id string) (*types.Job, error) {
	return scheduler.Jobs[id], nil
}

func (scheduler *Libp2pScheduler) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	scheduler.SubscribeFuncs = append(scheduler.SubscribeFuncs, subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pScheduler) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	jobUuid, err := uuid.NewRandom()
	if err != nil {
		return nil, fmt.Errorf("Error in creating job id. %s", err)
	}

	jobId := jobUuid.String()

	err = scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_CREATED,
		JobSpec:   spec,
		JobDeal:   deal,
	})

	if err != nil {
		return nil, err
	}

	job := &types.Job{
		Id:    jobId,
		Spec:  spec,
		Deal:  deal,
		State: make(map[string]*types.JobState),
	}

	return job, nil
}

func (scheduler *Libp2pScheduler) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_DEAL_UPDATED,
		JobDeal:   deal,
	})
}

func (scheduler *Libp2pScheduler) CancelJob(jobId string) error {
	return nil
}

func (scheduler *Libp2pScheduler) AcceptJobBid(jobId, nodeId string) error {
	deal := scheduler.Jobs[jobId].Deal
	deal.AssignedNodes = append(deal.AssignedNodes, nodeId)
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_BID_ACCEPTED,
		JobDeal:   deal,
		JobState: &types.JobState{
			State: system.JOB_STATE_RUNNING,
		},
	})
}

func (scheduler *Libp2pScheduler) RejectJobBid(jobId, nodeId, message string) error {
	if message == "" {
		message = "Job bid rejected by client"
	}
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_BID_REJECTED,
		JobState: &types.JobState{
			State:  system.JOB_STATE_BID_REJECTED,
			Status: message,
		},
	})
}

func (scheduler *Libp2pScheduler) AcceptResult(jobId, nodeId string) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_RESULTS_ACCEPTED,
		JobState: &types.JobState{
			State: system.JOB_STATE_RESULTS_ACCEPTED,
		},
	})
}

func (scheduler *Libp2pScheduler) RejectResult(jobId, nodeId, message string) error {
	if message == "" {
		message = "Job result rejected by client"
	}
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_RESULTS_REJECTED,
		JobState: &types.JobState{
			State:  system.JOB_STATE_RESULTS_REJECTED,
			Status: message,
		},
	})
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (scheduler *Libp2pScheduler) BidJob(jobId string) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_BID,
		JobState: &types.JobState{
			State: system.JOB_STATE_BIDDING,
		},
	})
}

func (scheduler *Libp2pScheduler) SubmitResult(jobId, status string, outputs []types.JobStorage) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_RESULTS,
		JobState: &types.JobState{
			State:   system.JOB_STATE_COMPLETE,
			Status:  status,
			Outputs: outputs,
		},
	})
}

func (scheduler *Libp2pScheduler) ErrorJob(jobId, status string) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		EventName: system.JOB_EVENT_ERROR,
		JobState: &types.JobState{
			State:  system.JOB_STATE_ERROR,
			Status: status,
		},
	})
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (scheduler *Libp2pScheduler) ErrorJobForNode(jobId, nodeId, status string) error {
	return scheduler.writeJobEvent(&types.JobEvent{
		JobId:     jobId,
		NodeId:    nodeId,
		EventName: system.JOB_EVENT_ERROR,
		JobState: &types.JobState{
			State:  system.JOB_STATE_ERROR,
			Status: status,
		},
	})
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

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

func (scheduler *Libp2pScheduler) writeJobEvent(event *types.JobEvent) error {
	if event.NodeId == "" {
		nodeId, err := scheduler.HostId()
		if err != nil {
			return fmt.Errorf("Error in creating job id. %s", err)
		}
		event.NodeId = nodeId
	}
	msgBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}
	log.Debug().Msgf("Sending event: %s\n", string(msgBytes))
	return scheduler.JobEventTopic.Publish(scheduler.Ctx, msgBytes)
}

func (scheduler *Libp2pScheduler) readLoopJobEvents() {
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

		// let's initialise the state for this job because it was just created
		if jobEvent.EventName == system.JOB_EVENT_CREATED {
			scheduler.Jobs[jobEvent.JobId] = &types.Job{
				Id:    jobEvent.JobId,
				Owner: jobEvent.NodeId,
				Spec:  nil,
				Deal:  nil,
				State: make(map[string]*types.JobState),
			}
		}

		// for "create" and "update" events - this will be filled in
		if jobEvent.JobSpec != nil {
			scheduler.Jobs[jobEvent.JobId].Spec = jobEvent.JobSpec
		}

		// only the owner of the job can update
		if jobEvent.JobDeal != nil {
			scheduler.Jobs[jobEvent.JobId].Deal = jobEvent.JobDeal
		}

		// both the jobState struct and the NodeId are required
		// because the job state is "against" the node
		if jobEvent.JobState != nil && jobEvent.NodeId != "" {
			scheduler.Jobs[jobEvent.JobId].State[jobEvent.NodeId] = jobEvent.JobState
		}

		for _, subscribeFunc := range scheduler.SubscribeFuncs {
			go subscribeFunc(jobEvent, scheduler.Jobs[jobEvent.JobId])
		}
	}
}
