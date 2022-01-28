package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

type ComputeNode struct {
	Ctx context.Context
	// the jobs we have already filtered and might want to process
	Jobs []Job
	// new jobs arriving via libp2p pubsub
	NewJobs      chan *Job
	Host         host.Host
	PubSub       *pubsub.PubSub
	Topic        *pubsub.Topic
	Subscription *pubsub.Subscription
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
	topic, err := pubsub.Join("bacalhau-jobs")
	if err != nil {
		return nil, err
	}
	subscription, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}
	server := &ComputeNode{
		Ctx:          ctx,
		Jobs:         []Job{},
		Host:         host,
		PubSub:       pubsub,
		Topic:        topic,
		Subscription: subscription,
	}
	go server.ReadLoop()
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

// this should be ctrl+c to exit
func (server *ComputeNode) Render() {
	fmt.Printf("we have %d jobs\n", len(server.Jobs))
	fmt.Printf("%+v\n", server.Jobs)
}

func (server *ComputeNode) AddJob(job *Job) {
	// TODO: filter the job - is this done async?

	// send valid messages onto the Messages channel
	server.Jobs = append(server.Jobs, *job)
	server.Render()

	// TODO: split this into an async thing that is working through the mempool
	server.RunJob(job)
}

func (server *ComputeNode) RunCommand(command string, args []string) error {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func (server *ComputeNode) RunCommandGetResults(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	result, err := cmd.CombinedOutput()
	return string(result), err
}

func (server *ComputeNode) RunJob(job *Job) error {

	localJobId := fmt.Sprintf("%s%d", job.Id, os.Getpid())

	wd, err := os.Getwd()

	if err != nil {
		return err
	}

	localTraceImagePath := fmt.Sprintf("%s/outputs/%s.png", wd, localJobId)

	fmt.Printf("GOT JOB!\n%+v\n", job)

	// start a firecracker VM
	// loop over each command - ignite exec <id> <command>
	err = server.RunCommand("sudo", []string{
		"ignite",
		"run",
		"weaveworks/ignite-ubuntu",
		"--name",
		localJobId,
		"--cpus",
		fmt.Sprintf("%d", job.Cpu),
		"--memory",
		fmt.Sprintf("%dGB", job.Memory),
		"--size",
		fmt.Sprintf("%dGB", job.Disk),
		"--ssh",
	})

	if err != nil {
		return err
	}

	// TODO: XXX SECURITY HOLE XXX (untrusted input feed to command execution string)
	pid, err := server.RunCommandGetResults("sudo", []string{
		"bash",
		"-c",
		fmt.Sprintf("sudo ps auxwwww |grep $(sudo ignite inspect vm %s | jq -r .metadata.uid) |grep 'firecracker --api-sock' |awk '{print $2}'", localJobId),
	})

	if err != nil {
		return err
	}

	fmt.Printf("IGNITE PID: %s\n", pid)

	for _, command := range job.BuildCommands {

		fmt.Printf("RUNNING BUILD COMMAND: %s\n", command)

		err = server.RunCommand("sudo", []string{
			"ignite",
			"exec",
			localJobId,
			command,
		})

		if err != nil {
			return err
		}
	}

	fmt.Printf("CREATING OUTPUT FOLDER: \n")

	err = server.RunCommand("mkdir", []string{
		"-p",
		"outputs",
	})

	if err != nil {
		return err
	}

	// start a monitoring process for the job
	traceCmd := exec.Command("sudo", []string{
		"psrecord",
		pid,
		"--plot",
		localTraceImagePath,
	}...)

	err = traceCmd.Start()

	if err != nil {
		return err
	}

	tracePid := traceCmd.Process.Pid
	fmt.Printf("TRACE PID: %d -> %s\n", tracePid, localTraceImagePath)

	for _, command := range job.Commands {

		fmt.Printf("RUNNING COMMAND: %s\n", command)

		err = server.RunCommand("sudo", []string{
			"ignite",
			"exec",
			localJobId,
			command,
		})

		if err != nil {
			return err
		}
	}

	fmt.Printf("STOP TRACE PID: %d\n", tracePid)
	traceCmd.Process.Signal(syscall.SIGINT)

	return nil
}

func (server *ComputeNode) ReadLoop() {
	for {
		msg, err := server.Subscription.Next(server.Ctx)
		if err != nil {
			close(server.NewJobs)
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == server.Host.ID() {
			continue
		}
		job := new(Job)
		err = json.Unmarshal(msg.Data, job)
		if err != nil {
			continue
		}
		go server.AddJob(job)
	}
}

func (server *ComputeNode) Publish(job *Job) error {
	msgBytes, err := json.Marshal(job)
	if err != nil {
		return err
	}
	server.AddJob(job)
	return server.Topic.Publish(server.Ctx, msgBytes)
}
