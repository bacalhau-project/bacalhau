package internal

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/filecoin-project/bacalhau/internal/ipfs"
	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/filecoin-project/bacalhau/internal/runtime"
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
	Id                      string
	IpfsRepo                string
	IpfsConnectMultiAddress string

	Ctx context.Context

	// the jobs we have already filtered and might want to process
	Jobs map[string]*types.JobData

	Host                  host.Host
	PubSub                *pubsub.PubSub
	JobCreateTopic        *pubsub.Topic
	JobCreateSubscription *pubsub.Subscription
	JobUpdateTopic        *pubsub.Topic
	JobUpdateSubscription *pubsub.Subscription
}

func PrettyPrintComputeNode(cn *ComputeNode) string {
	jobString, _ := json.MarshalIndent(cn.Jobs, "", "  ")
	return fmt.Sprintf(`
	Id: %s
	IpfsRepo: %s
	IpfsConnectMultiAddress: %s 
	Ctx: %s
	Jobs: %v
	Host: %s
	JobCreateTopic: %s
	JobCreateSubscription: %s
	JobUpdateTopic: %s
	JobUpdateSubscription: %s
`, cn.Id, cn.IpfsRepo, cn.IpfsConnectMultiAddress, cn.Ctx, string(jobString), cn.Host, cn.JobCreateTopic.String(), cn.JobCreateSubscription.Topic(), cn.JobUpdateTopic.String(), cn.JobUpdateSubscription.Topic())

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
		Id:                      host.ID().String(),
		IpfsRepo:                "",
		IpfsConnectMultiAddress: "",
		Ctx:                     ctx,
		Jobs:                    make(map[string]*types.JobData),
		Host:                    host,
		PubSub:                  pubsub,
		JobCreateTopic:          jobCreateTopic,
		JobCreateSubscription:   jobCreateSubscription,
		JobUpdateTopic:          jobUpdateTopic,
		JobUpdateSubscription:   jobUpdateSubscription,
	}
	go server.ReadLoopJobCreate()
	go server.ReadLoopJobUpdate()
	go func() {
		logger.Debug("waiting for bacalhau libp2p context done\n")
		<-ctx.Done()
		logger.Debug("closing bacalhau libp2p daemon\n")
		host.Close()
		logger.Debug("closed bacalhau libp2p daemon\n")
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

	//nolint
	server.Host.Connect(server.Ctx, *info)

	return nil
}

func (server *ComputeNode) FilterJob(job *types.JobSpec) (bool, error) {
	logger.Debugf("Testing to see whether or not this server (Id: %s) should accept job Id (%s) with CIDs (%s)", server.Id, job.Id, strings.Join(job.Cids, ", "))
	// Accept jobs where there are no cids specified or we have any one of the specified cids
	if len(job.Cids) == 0 {
		logger.Infof("	No CIDs selected, so all jobs apply. Accepting job id: %s", job.Id)
		return true, nil
	}
	logger.Debugf("Job has CIDs, so determining if CIDs are on this server.")
	for _, cid := range job.Cids {
		hasCid, err := ipfs.HasCid(server.IpfsRepo, cid)
		if err != nil {
			return false, err
		}
		if hasCid {
			logger.Infof("CID (%s) found on this server (Id: %s). Accepting job id: %s", cid, server.Id, job.Id)
			return true, nil
		}
	}

	logger.Infof("No matching CIDs found on this server (Id: %s). Passing on job (%s)", server.Id, job.Id)
	return false, nil
}

func (server *ComputeNode) AddJob(job *types.JobSpec) {

	// add the job to the mempool of all nodes because then we can ask a question "list the jobs"
	// TODO: this is not efficient but it's state that we can use in the CLI
	server.Jobs[job.Id] = &types.JobData{
		Job:   job,
		State: make(map[string]*types.JobState),
	}

	jobsArray, _ := json.MarshalIndent(server.Jobs, "", "  ")
	logger.Infof("Determining whether or not Job (Id: %s) is a fit for Server (Id: %s)", job.Id, server.Id)
	logger.Debug("Jobs on this server: %d", len(server.Jobs), string(jobsArray))

	shouldRunJob, err := server.FilterJob(job)

	if err != nil {
		logger.Errorf("Error in self selecting: %s\n%+v\n", err, job)
		return
	}

	if !shouldRunJob {
		logger.Infof("Server %s did not select job with id: %s", server.Id, job.Id)
		return
	}

	// TODO: split this into an async thing that is working through the mempool
	logger.Infof("-----------")
	logger.Infof("Server %s DID select job: %s", server.Id, job.Id)
	logger.Infof("-----------")

	// update the network with the fact that we have selected the job
	err = server.ChangeJobState(
		job,
		"selected",
		fmt.Sprintf("Job was selected because jobs CID are local:\n %+v\n", job.Cids),
		"",
	)

	if err != nil {
		logger.Errorf("Server %s could not change job (Id: %s) state to SELECTED. Error: %s", server.Id, job.Id, err)
		return
	}

	cid, err := server.RunJob(job)

	if err != nil {
		logger.Errorf("Server %s could not run job (Id: %s). Error: %s", server.Id, job.Id, err)

		err = server.ChangeJobState(
			job,
			"error",
			fmt.Sprintf("Error running the job (Id: %s): %s", job.Id, err),
			"",
		)

		if err != nil {
			logger.Errorf("Server %s could not change job (Id: %s) state to ERROR. Error: %s", server.Id, job.Id, err)
			return
		}

		return
	}

	//nolint
	err = server.ChangeJobState(
		job,
		"complete",
		fmt.Sprintf("Job (Id: %s) is now complete", job.Id),
		cid,
	)

	if err != nil {
		logger.Errorf("Server %s could not change job (Id: %s) state to COMPLETE. Error: %s", server.Id, job.Id, err)
		return
	}
}

func (server *ComputeNode) UpdateJob(update *types.JobState) {
	logger.Infof("Server (Id: %s) updating job (Id: %s). Was: %s\tNow: %s", server.Id, update.JobId, server.Jobs[update.JobId].State, update.State)
	server.Jobs[update.JobId].State[update.NodeId] = update
}

// return a CID of the job results when finished
// this is obtained by running "ipfs add -r <results folder>"
func (server *ComputeNode) RunJob(job *types.JobSpec) (string, error) {

	vm, err := runtime.NewRuntime(job)

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

	//nolint
	defer vm.Stop()

	err = vm.PrepareJob(server.IpfsConnectMultiAddress)

	if err != nil {
		return "", err
	}

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
		job := new(types.JobSpec)
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
		jobUpdate := new(types.JobState)
		err = json.Unmarshal(msg.Data, jobUpdate)
		if err != nil {
			continue
		}
		go server.UpdateJob(jobUpdate)

		// emit the event to listeners
	}
}

func (server *ComputeNode) Publish(job *types.JobSpec) error {
	logger.Infof(`
New job registered: %s

`, types.PrettyPrintJob(job))

	msgBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	go server.AddJob(job)
	logger.Debugf("Server Details:\n%s", PrettyPrintComputeNode(server))
	ctx := server.Ctx
	topic := server.JobCreateTopic
	return topic.Publish(ctx, msgBytes)
}

func (server *ComputeNode) ChangeJobState(job *types.JobSpec, state, status, resultCid string) error {
	update := &types.JobState{
		JobId:     job.Id,
		NodeId:    server.Id,
		State:     state,
		Status:    status,
		ResultCid: resultCid,
	}
	msgBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	go server.UpdateJob(update)
	return server.JobUpdateTopic.Publish(server.Ctx, msgBytes)
}
