package libp2p

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/filecoin-project/bacalhau/pkg/system"
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
	// Lifecycle context for the transport.
	ctx context.Context

	// Writer we emit events through.
	genericTransport     *transport.GenericTransport
	Host                 host.Host
	Port                 int
	PubSub               *pubsub.PubSub
	JobEventTopic        *pubsub.Topic
	JobEventSubscription *pubsub.Subscription
}

func getConfigPath() string {
	suffix := "/.bacalhau"
	env := os.Getenv("BACALHAU_PATH")
	var d string
	if env == "" {
		// e.g. /home/francesca/.bacalhau
		dirname, err := os.UserHomeDir()
		if err != nil {
			panic(err)
		}
		d = dirname + suffix
	} else {
		// e.g. /data/.bacalhau
		d = env + suffix
	}
	// create dir if not exists
	if err := os.MkdirAll(d, 0700); err != nil {
		panic(err)
	}
	return d
}

func makeLibp2pHost(
	port int,
) (host.Host, error) {

	configPath := getConfigPath()
	// We include the port in the filename so that in devstack multiple nodes
	// running on the same host get different identities
	privKeyPath := fmt.Sprintf("%s/private_key.%d", configPath, port)

	if _, err := os.Stat(privKeyPath); errors.Is(err, os.ErrNotExist) {
		// Private key does not exist - create and write it

		// Creates a new RSA key pair for this host.
		prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, rand.Reader)
		if err != nil {
			log.Error().Err(err)
			return nil, err
		}

		keyOut, err := os.OpenFile(privKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return nil, fmt.Errorf("failed to open key.pem for writing: %v", err)
		}
		privBytes, err := crypto.MarshalPrivateKey(prvKey)
		if err != nil {
			return nil, fmt.Errorf("unable to marshal private key: %v", err)
		}
		// base64 encode privBytes
		b64 := base64.StdEncoding.EncodeToString(privBytes)
		_, err = keyOut.Write([]byte(b64 + "\n"))
		if err != nil {
			return nil, fmt.Errorf("failed to write to key file: %v", err)
		}
		if err := keyOut.Close(); err != nil {
			return nil, fmt.Errorf("error closing key file: %v", err)
		}
		log.Printf("wrote %s", privKeyPath)
	}

	// Now that we've ensured the private key is written to disk, read it! This
	// ensures that loading it works even in the case where we've just created
	// it.

	// read the private key
	keyBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %v", err)
	}
	// base64 decode keyBytes
	b64, err := base64.StdEncoding.DecodeString(string(keyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %v", err)
	}
	// parse the private key
	prvKey, err := crypto.UnmarshalPrivateKey(b64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
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

func NewLibp2pTransport(ctx context.Context, port int) (
	*Libp2pTransport, error) {

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
		ctx:                  ctx,
		Host:                 host,
		Port:                 port,
		PubSub:               pubsub,
		JobEventTopic:        jobEventTopic,
		JobEventSubscription: jobEventSubscription,
	}

	// setup the event writer
	libp2pTransport.genericTransport = transport.NewGenericTransport(
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

func (transport *Libp2pTransport) HostId() (string, error) {
	return transport.Host.ID().String(), nil
}

func (transport *Libp2pTransport) Start() error {
	if len(transport.genericTransport.SubscribeFuncs) <= 0 {
		panic("Programming error: no subscribe func, please call Subscribe immediately after constructing interface")
	}

	go transport.readLoopJobEvents()
	log.Debug().Msg("Libp2p transport has started")

	system.OnCancel(transport.ctx, func() {
		transport.Host.Close()
		log.Debug().Msg("Libp2p transport has stopped")
	})

	return nil
}

/////////////////////////////////////////////////////////////
/// READ OPERATIONS
/////////////////////////////////////////////////////////////

func (transport *Libp2pTransport) List() (types.ListResponse, error) {
	return transport.genericTransport.List()
}

func (transport *Libp2pTransport) Get(id string) (*types.Job, error) {
	return transport.genericTransport.Get(id)
}

func (transport *Libp2pTransport) Subscribe(subscribeFunc func(jobEvent *types.JobEvent, job *types.Job)) {
	transport.genericTransport.Subscribe(subscribeFunc)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "CLIENT" / REQUESTER
/////////////////////////////////////////////////////////////

func (transport *Libp2pTransport) SubmitJob(spec *types.JobSpec, deal *types.JobDeal) (*types.Job, error) {
	return transport.genericTransport.SubmitJob(spec, deal)
}

func (transport *Libp2pTransport) UpdateDeal(jobId string, deal *types.JobDeal) error {
	return transport.genericTransport.UpdateDeal(jobId, deal)
}

func (transport *Libp2pTransport) CancelJob(jobId string) error {
	return nil
}

func (transport *Libp2pTransport) AcceptJobBid(jobId, nodeId string) error {
	return transport.genericTransport.AcceptJobBid(jobId, nodeId)
}

func (transport *Libp2pTransport) RejectJobBid(jobId, nodeId, message string) error {
	return transport.genericTransport.RejectJobBid(jobId, nodeId, message)
}

/////////////////////////////////////////////////////////////
/// WRITE OPERATIONS - "SERVER" / COMPUTE NODE
/////////////////////////////////////////////////////////////

func (transport *Libp2pTransport) BidJob(jobId string) error {
	return transport.genericTransport.BidJob(jobId)
}

func (transport *Libp2pTransport) SubmitResult(jobId, status, resultsId string) error {
	return transport.genericTransport.SubmitResult(jobId, status, resultsId)
}

func (transport *Libp2pTransport) ErrorJob(jobId, status string) error {
	return transport.genericTransport.ErrorJob(jobId, status)
}

// this is when the requester node needs to error the status for a node
// for example - results have been given by the compute node
// and in checking the results, the requester node came across some kind of error
// we need to flag that error against the node that submitted the results
// (but we are the requester node) - so we need this util function
func (transport *Libp2pTransport) ErrorJobForNode(jobId, nodeId, status string) error {
	return transport.genericTransport.ErrorJobForNode(jobId, nodeId, status)
}

/////////////////////////////////////////////////////////////
/// INTERNAL IMPLEMENTATION
/////////////////////////////////////////////////////////////

func (transport *Libp2pTransport) Connect(peerConnect string) error {

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

	transport.Host.Peerstore().AddAddrs(
		info.ID, info.Addrs, peerstore.PermanentAddrTTL)

	return transport.Host.Connect(transport.ctx, *info)
}

func (transport *Libp2pTransport) writeJobEvent(event *types.JobEvent) error {
	bs, err := json.Marshal(event)
	if err != nil {
		return err
	}

	log.Debug().Msgf("Sending event: %s", string(bs))
	return transport.JobEventTopic.Publish(transport.ctx, bs)
}

func (transport *Libp2pTransport) readLoopJobEvents() {
	for {
		msg, err := transport.JobEventSubscription.Next(transport.ctx)
		if err != nil {
			return
		}

		jobEvent := new(types.JobEvent)
		err = json.Unmarshal(msg.Data, jobEvent)
		if err != nil {
			continue
		}

		transport.genericTransport.ReadEvent(jobEvent)
	}
}
