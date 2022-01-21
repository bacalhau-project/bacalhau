package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
	for {
		fmt.Printf("we have %d jobs\n", len(server.Jobs))
		fmt.Printf("%+v\n", server.Jobs)
		time.Sleep(time.Second * 1)
	}
}

func (server *ComputeNode) AddJob(job *Job) {
	// TODO: filter the job

	// send valid messages onto the Messages channel
	server.Jobs = append(server.Jobs, *job)
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
		server.AddJob(job)
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
