package libp2p

import (
	"context"
	"testing"
	"time"

	libp2p_host "github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	libp2p_pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

const testTopic = "test-topic"

type PubSubSuite struct {
	suite.Suite
	node1       *PubSub[string]
	node2       *PubSub[string]
	subscriber1 *pubsub.InMemorySubscriber[string]
	subscriber2 *pubsub.InMemorySubscriber[string]
}

func (s *PubSubSuite) SetupSuite() {
	n1, h1 := s.createPubSub(false)
	s.node1 = n1
	s.node2, _ = s.createPubSub(true, h1)

	s.subscriber1 = pubsub.NewInMemorySubscriber[string]()
	s.subscriber2 = pubsub.NewInMemorySubscriber[string]()
	s.NoError(s.node1.Subscribe(context.Background(), s.subscriber1))
	s.NoError(s.node2.Subscribe(context.Background(), s.subscriber2))

	// wait for up to 10 seconds (5 loops with 2 seconds each) for nodes to discover each other
	var s1, s2 bool
	for i := 0; i < 5; i++ {
		s.NoError(s.node1.Publish(context.Background(), "ping"))
		s1, s2 = s.waitForMessage("ping", 2*time.Second, true, true)
		if s1 || s2 {
			// still one of the subscribers is waiting for the message
			continue
		}
	}
	if s1 {
		s.FailNow("subscriber 1 didn't receive initialization message")
	}
	if s2 {
		s.FailNow("subscriber 2 didn't receive initialization message")
	}
	log.Debug().Msg("libp2p pubsub suite is ready")
}

func (s *PubSubSuite) TearDownSuite() {
	s.NoError(s.node1.Close(context.Background()))
	s.NoError(s.node2.Close(context.Background()))
}

func (s *PubSubSuite) createPubSub(ignoreLocal bool, peers ...host.Host) (*PubSub[string], host.Host) {
	h, err := libp2p_host.NewHostForTest(context.Background(), peers...)
	s.NoError(err)

	gossipSub, err := libp2p_pubsub.NewGossipSub(context.Background(), h)
	s.NoError(err)

	pubSub, err := NewPubSub[string](PubSubParams{
		Host:        h,
		TopicName:   testTopic,
		PubSub:      gossipSub,
		IgnoreLocal: ignoreLocal,
	})
	s.NoError(err)

	return pubSub, h
}

func TestPubSubSuite(t *testing.T) {
	suite.Run(t, new(PubSubSuite))
}

func (s *PubSubSuite) TestPubSub() {
	msg := "TestPubSub"
	s.NoError(s.node1.Publish(context.Background(), msg))
	s.waitForMessage(msg, 10*time.Second, true, true)
}

func (s *PubSubSuite) TestPubSub_IgnoreLocal() {
	// node2 is ignoring local messages, so it should not receive the message
	msg := "TestPubSub_IgnoreLocal"
	s.NoError(s.node2.Publish(context.Background(), msg))
	s.waitForMessage(msg, 10*time.Second, true, false)
	s.Empty(s.subscriber2.Events())
}

func (s *PubSubSuite) waitForMessage(msg string, duration time.Duration, checkSubscriber1, checkSubscriber2 bool) (bool, bool) {
	waitUntil := time.Now().Add(duration)
	checkSubscriber := func(subscriber *pubsub.InMemorySubscriber[string]) bool {
		events := subscriber.Events()
		if len(events) == 0 {
			return false
		}
		s.Equal([]string{msg}, events)
		return true
	}

	for time.Now().Before(waitUntil) && (checkSubscriber1 || checkSubscriber2) {
		time.Sleep(100 * time.Millisecond)
		if checkSubscriber1 && checkSubscriber(s.subscriber1) {
			checkSubscriber1 = false
		}
		if checkSubscriber2 && checkSubscriber(s.subscriber2) {
			checkSubscriber2 = false
		}
	}

	return checkSubscriber1, checkSubscriber1
}
