//go:build unit || !integration

package pubsub

import (
	"context"
	"testing"
	"time"

	nats_helper "github.com/bacalhau-project/bacalhau/pkg/nats"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/suite"
)

const subjectName = "topic.greetings"

type PubSubSuite struct {
	suite.Suite
	natsServer  *server.Server
	node1       *PubSub[string]
	node2       *PubSub[string]
	subscriber1 *pubsub.InMemorySubscriber[string]
	subscriber2 *pubsub.InMemorySubscriber[string]
}

func (s *PubSubSuite) SetupSuite() {
	ctx := context.Background()
	s.natsServer = s.createNatsServer()
	s.node1 = s.createPubSub(ctx, subjectName, "", s.natsServer.ClientURL())
	s.node2 = s.createPubSub(ctx, subjectName, "topic.*", s.natsServer.ClientURL())

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
	log.Debug().Msg("nats pubsub suite is ready")
}

func (s *PubSubSuite) TearDownSuite() {
	s.NoError(s.node1.Close(context.Background()))
	s.NoError(s.node2.Close(context.Background()))
}

// createNatsServer creates a new nats server
func (s *PubSubSuite) createNatsServer() *server.Server {
	ctx := context.Background()
	port, err := freeport.GetFreePort()
	s.Require().NoError(err)

	serverOpts := server.Options{
		Port: port,
	}

	ns, err := nats_helper.NewServerManager(ctx, &serverOpts)
	s.Require().NoError(err)

	return ns.Server
}

func (s *PubSubSuite) createPubSub(ctx context.Context, subject, subscriptionSubject string, server string) *PubSub[string] {
	clientManager, err := nats_helper.NewClientManager(ctx, nats_helper.ClientManagerParams{
		Name:    "test",
		Servers: server,
	})
	s.Require().NoError(err)

	pubSub, err := NewPubSub[string](PubSubParams{
		Conn:                clientManager.Client,
		Subject:             subject,
		SubscriptionSubject: subscriptionSubject,
	})
	s.Require().NoError(err)

	return pubSub
}

func TestPubSubSuite(t *testing.T) {
	suite.Run(t, new(PubSubSuite))
}

func (s *PubSubSuite) TestPubSub() {
	msg := "TestPubSub"
	s.NoError(s.node1.Publish(context.Background(), msg))
	s.waitForMessage(msg, 10*time.Second, true, true)
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
