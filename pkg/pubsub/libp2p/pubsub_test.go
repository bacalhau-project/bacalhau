package libp2p

import (
	"context"
	"testing"
	"time"

	libp2p_host "github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"

	"github.com/stretchr/testify/suite"
)

const testTopic = "test-topic"

type PubSubSuite struct {
	suite.Suite
	node1       *PubSub[string]
	node2       *PubSub[string]
	subscriber1 *pubsub.InMemorySubscriber[string]
	subscriber2 *pubsub.InMemorySubscriber[string]
	ignoreLocal bool
}

func (s *PubSubSuite) SetupTest() {
	s.subscriber1 = pubsub.NewInMemorySubscriber[string]()
	s.subscriber2 = pubsub.NewInMemorySubscriber[string]()
}

func (s *PubSubSuite) TearDownTest() {
	s.NoError(s.node1.Close(context.Background()))
	s.NoError(s.node2.Close(context.Background()))
}

func (s *PubSubSuite) setupHosts() {
	n1, h1 := s.createPubSub()
	s.node1 = n1
	s.node2, _ = s.createPubSub(h1)
	s.NoError(s.node1.Subscribe(context.Background(), s.subscriber1))
	s.NoError(s.node2.Subscribe(context.Background(), s.subscriber2))

}

func (s *PubSubSuite) createPubSub(peers ...host.Host) (*PubSub[string], host.Host) {
	port, err := freeport.GetFreePort()
	s.NoError(err)

	h, err := libp2p_host.NewHost(port)
	s.NoError(err)

	libp2pPeer := []multiaddr.Multiaddr{}
	for _, peer := range peers {
		for _, addrs := range peer.Addrs() {
			p2pAddr, p2pAddrErr := multiaddr.NewMultiaddr("/p2p/" + peer.ID().String())
			s.NoError(p2pAddrErr)
			libp2pPeer = append(libp2pPeer, addrs.Encapsulate(p2pAddr))
		}
	}
	if len(libp2pPeer) > 0 {
		s.NoError(libp2p_host.ConnectToPeers(context.Background(), h, libp2pPeer))
	}

	gossipSub, err := libp2p_pubsub.NewGossipSub(context.Background(), h)
	s.NoError(err)

	pubSub := NewPubSub[string](PubSubParams{
		Host:        h,
		TopicName:   testTopic,
		PubSub:      gossipSub,
		IgnoreLocal: s.ignoreLocal,
	})
	s.NoError(err)

	return pubSub, h
}

func TestPubSubSuite(t *testing.T) {
	suite.Run(t, new(PubSubSuite))
}

func (s *PubSubSuite) TestPubSub() {
	s.setupHosts()

	// wait for nodes to discover each other
	time.Sleep(1 * time.Second)

	// wait for nodes to discover each other
	s.NoError(s.node1.Publish(context.Background(), "hello"))

	// wait for message to be published to other nodes
	time.Sleep(1 * time.Second)
	s.Equal([]string{"hello"}, s.subscriber1.Events())
	s.Equal([]string{"hello"}, s.subscriber2.Events())
}

func (s *PubSubSuite) TestPubSub_IgnoreLocal() {
	s.ignoreLocal = true
	s.setupHosts()

	// wait for nodes to discover each other
	time.Sleep(1 * time.Second)

	// publish message
	s.NoError(s.node1.Publish(context.Background(), "hello"))

	// wait for message to be published to other nodes
	time.Sleep(1 * time.Second)
	s.Equal([]string{}, s.subscriber1.Events())
	s.Equal([]string{"hello"}, s.subscriber2.Events())
}
