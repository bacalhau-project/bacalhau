package internal

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"

	"github.com/filecoin-project/bacalhau/internal/ignite"
	"github.com/filecoin-project/bacalhau/internal/ipfs"
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

const IGNITE_IMAGE string = "docker.io/binocarlos/bacalhau-ignite-image:latest"

type ComputeNode struct {
	Id       string
	IpfsRepo string
	Ctx      context.Context
	// the jobs we have already filtered and might want to process
	Jobs []types.Job

	// see types.Update for the message that updates these fields

	// a map of job id into bacalhau nodes that are in progress of doing some work with their claimed state and human readable statuses
	JobState  map[string]map[string]string
	JobStatus map[string]map[string]string
	// a map of job id onto bacalhau compute nodes that claim to have done the work onto cids of the job results published by them
	JobResults map[string]map[string]string

	// are we using a temporary IPFS repo for local testing?
	TempIpfsRepo          bool
	Host                  host.Host
	PubSub                *pubsub.PubSub
	JobCreateTopic        *pubsub.Topic
	JobCreateSubscription *pubsub.Subscription
	JobUpdateTopic        *pubsub.Topic
	JobUpdateSubscription *pubsub.Subscription
}

func makeLibp2pHost(port int) (host.Host, error) {
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

func NewComputeNode(
	ctx context.Context,
	port int,
) (*ComputeNode, error) {
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
	server := &ComputeNode{
		Id:                    host.ID().String(),
		IpfsRepo:              "",
		Ctx:                   ctx,
		Jobs:                  []types.Job{},
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
	go server.ReadLoopJobCreate()
	go server.ReadLoopJobUpdate()
	go func() {
		<-ctx.Done()
		// jobCreateSubscription.Cancel()
		// jobUpdateSubscription.Cancel()
		host.Close()
	}()
	return server, nil
}

func (server *ComputeNode) Connect(peerConnect string) error {

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

	server.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	server.Host.Connect(server.Ctx, *info)

	return nil
}

func (server *ComputeNode) FilterJob(job *types.Job) (bool, error) {
	// Accept jobs where there are no cids specified or we have any one of the specified cids
	if len(job.Cids) == 0 {
		return true, nil
	}
	for _, cid := range job.Cids {
		hasCid, err := ipfs.HasCid(server.IpfsRepo, cid)
		if err != nil {
			return false, err
		}
		if hasCid {
			return true, nil
		}
	}

	return false, nil
}

func (server *ComputeNode) AddJob(job *types.Job) {

	// add the job to the mempool of all nodes because then we can ask a question "list the jobs"
	// TODO: this is not efficient but it's state that we can use in the CLI
	server.Jobs = append(server.Jobs, *job)
	fmt.Printf("we have %d jobs\n", len(server.Jobs))
	fmt.Printf("%+v\n", server.Jobs)

	shouldRunJob, err := server.FilterJob(job)

	if err != nil {
		fmt.Printf("there was an error self selecting: %s\n%+v\n", err, job)
		return
	}

	if !shouldRunJob {
		fmt.Printf("we ignored a job self selecting: \n%+v\n", job)
		return
	}

	// TODO: split this into an async thing that is working through the mempool
	fmt.Printf("we are running a job!: \n%+v\n", job)

	// update the network with the fact that we have selected the job
	err = server.ChangeJobState(
		job,
		"selected",
		fmt.Sprintf("Job was selected because jobs CID are local:\n %+v\n", job.Cids),
		"",
	)

	if err != nil {
		fmt.Printf("there was an error changing job state: %s\n%+v\n", err, job)
		return
	}

	cid, err := server.RunJob(job)

	if err != nil {
		fmt.Printf("there was an error running the job: %s\n%+v\n", err, job)

		err = server.ChangeJobState(
			job,
			"error",
			fmt.Sprintf("Error running the job: %s\n", err),
			"",
		)

		if err != nil {
			fmt.Printf("there was an error changing job state: %s\n%+v\n", err, job)
		}

		return
	}

	fmt.Printf("-------------\n\nCID: %s\n\n", cid)

	err = server.ChangeJobState(
		job,
		"complete",
		fmt.Sprintf("Job is now complete\n"),
		cid,
	)

	if err != nil {
		fmt.Printf("there was an error changing job state: %s\n%+v\n", err, job)
	}
}

func (server *ComputeNode) UpdateJob(update *types.Update) {
	fmt.Printf("we are updating a job!: \n%+v\n", update)

	if server.JobState[update.JobId] == nil {
		server.JobState[update.JobId] = make(map[string]string)
	}

	if server.JobStatus[update.JobId] == nil {
		server.JobStatus[update.JobId] = make(map[string]string)
	}

	if server.JobResults[update.JobId] == nil {
		server.JobResults[update.JobId] = make(map[string]string)
	}

	server.JobState[update.JobId][update.NodeId] = update.State
	server.JobStatus[update.JobId][update.NodeId] = update.Status
	server.JobResults[update.JobId][update.NodeId] = update.Output
}

// return a CID of the job results when finished
// this is obtained by running "ipfs add -r <results folder>"
func (server *ComputeNode) RunJob(job *types.Job) (string, error) {

	vm, err := ignite.NewVm(job)

	if err != nil {
		return "", err
	}

	resultsFolder, err := system.EnsureSystemDirectory(system.GetResultsDirectory(job.Id, server.Id))
	if err != nil {
		return "", err
	}

	err = vm.Start()

	if err != nil {
		return "", err
	}

	defer vm.Stop()

	err = vm.RunJob(resultsFolder)

	if err != nil {
		return "", err
	}

	resultCid, err := ipfs.AddFolder(server.IpfsRepo, resultsFolder)

	if err != nil {
		return "", err
	}

	return resultCid, nil
}

func (server *ComputeNode) ReadLoopJobCreate() {
	for {
		msg, err := server.JobCreateSubscription.Next(server.Ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == server.Host.ID() {
			continue
		}
		job := new(types.Job)
		err = json.Unmarshal(msg.Data, job)
		if err != nil {
			continue
		}
		go server.AddJob(job)
	}
}

func (server *ComputeNode) ReadLoopJobUpdate() {
	for {
		msg, err := server.JobUpdateSubscription.Next(server.Ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == server.Host.ID() {
			continue
		}
		jobUpdate := new(types.Update)
		err = json.Unmarshal(msg.Data, jobUpdate)
		if err != nil {
			continue
		}
		go server.UpdateJob(jobUpdate)
	}
}

func (server *ComputeNode) Publish(job *types.Job) error {
	msgBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	go server.AddJob(job)
	return server.JobCreateTopic.Publish(server.Ctx, msgBytes)
}

func (server *ComputeNode) ChangeJobState(job *types.Job, state, status, output string) error {
	update := &types.Update{
		JobId:  job.Id,
		NodeId: server.Id,
		State:  state,
		Status: status,
		Output: output,
	}
	msgBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	go server.UpdateJob(update)
	return server.JobUpdateTopic.Publish(server.Ctx, msgBytes)
}
