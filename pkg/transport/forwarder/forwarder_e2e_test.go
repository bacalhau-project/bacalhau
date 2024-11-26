//go:build unit || !integration

package forwarder_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/pkg/transport"
	"github.com/bacalhau-project/bacalhau/pkg/transport/forwarder"
)

type ForwarderE2ETestSuite struct {
	suite.Suite
	ctx          context.Context
	cancel       context.CancelFunc
	natsServer   *server.Server
	nc           *nats.Conn
	store        watcher.EventStore
	registry     *envelope.Registry
	subscriber   ncl.Subscriber
	received     []*envelope.Message
	cleanupFuncs []func()
}

func (s *ForwarderE2ETestSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Start NATS server
	s.natsServer, s.nc = testutils.StartNats(s.T())

	// Create boltdb store and watcher
	eventObjectSerializer := watcher.NewJSONSerializer()
	s.Require().NoError(eventObjectSerializer.RegisterType("test", reflect.TypeOf("")))
	store, err := boltdb.NewEventStore(
		watchertest.CreateBoltDB(s.T()),
		boltdb.WithEventSerializer(eventObjectSerializer),
	)
	s.Require().NoError(err)
	s.store = store

	// Create registry
	s.registry = envelope.NewRegistry()
	s.Require().NoError(s.registry.Register("test", "string"))

	// Create subscriber
	s.received = make([]*envelope.Message, 0)
	var msgHandler ncl.MessageHandlerFunc = func(_ context.Context, msg *envelope.Message) error {
		payload, _ := msg.GetPayload("string")
		s.T().Logf("Received message: %s", payload)
		s.received = append(s.received, msg)
		return nil
	}
	subscriber, err := ncl.NewSubscriber(s.nc, ncl.SubscriberConfig{
		Name:              "test-subscriber",
		MessageRegistry:   s.registry,
		MessageSerializer: envelope.NewSerializer(),
		MessageHandler:    msgHandler,
	})
	s.Require().NoError(err)
	s.subscriber = subscriber
	s.Require().NoError(s.subscriber.Subscribe(s.ctx, "test"))
}

func (s *ForwarderE2ETestSuite) TearDownTest() {
	for i := len(s.cleanupFuncs) - 1; i >= 0; i-- {
		s.cleanupFuncs[i]()
	}
	s.cleanupFuncs = nil

	if s.subscriber != nil {
		s.Require().NoError(s.subscriber.Close(context.Background()))
	}
	if s.nc != nil {
		s.nc.Close()
	}
	if s.natsServer != nil && s.natsServer.Running() {
		s.natsServer.Shutdown()
	}
	s.cancel()
}

func (s *ForwarderE2ETestSuite) startForwarder() *forwarder.Forwarder {
	w, err := watcher.New(s.ctx, "test-watcher", s.store)
	s.Require().NoError(err)

	publisher, err := ncl.NewOrderedPublisher(s.nc, ncl.OrderedPublisherConfig{
		Name:              "test-publisher",
		Destination:       "test",
		MessageRegistry:   s.registry,
		MessageSerializer: envelope.NewSerializer(),
		AckMode:           ncl.NoAck,
	})
	s.Require().NoError(err)

	f, err := forwarder.New(publisher, w, &testMessageCreator{})
	s.Require().NoError(err)

	s.Require().NoError(f.Start(s.ctx))

	s.cleanupFuncs = append(s.cleanupFuncs, func() {
		s.Require().NoError(f.Stop(s.ctx))
		s.Require().NoError(publisher.Close(s.ctx))
	})

	return f
}

func (s *ForwarderE2ETestSuite) TestEventFlow() {
	// Create forwarder
	s.startForwarder()

	// Store some events
	s.storeEvents(5)

	// Wait for processing
	s.Eventually(func() bool {
		return len(s.received) == 5
	}, time.Second, 10*time.Millisecond)
	s.Require().Equal(5, len(s.received))

	// Verify messages were published in order
	for i, msg := range s.received {
		s.verifyMsg(msg, i+1)
	}
}

func (s *ForwarderE2ETestSuite) TestReconnection() {
	// Create forwarder
	s.startForwarder()

	// Store an event
	s.storeEvent(1)

	// Wait for event to be published
	s.Eventually(func() bool {
		return len(s.received) == 1
	}, time.Second, 10*time.Millisecond)

	// Verify first message
	s.Require().Lenf(s.received, 1, "received: %v", s.received)
	s.verifyMsg(s.received[0], 1)

	// Stop NATS server
	s.natsServer.Shutdown()
	s.natsServer.WaitForShutdown()

	// wait for client to be disconnected
	s.Eventually(func() bool {
		return !s.nc.IsConnected()
	}, time.Second, 10*time.Millisecond)

	// Store another event - should not be lost
	s.storeEvent(2)

	// Restart NATS server
	s.natsServer, _ = testutils.RestartNatsServer(s.T(), s.natsServer)
	s.Eventually(func() bool {
		return s.nc.IsConnected()
	}, 5*time.Second, 10*time.Millisecond)

	// Store another event after reconnection
	s.storeEvent(3)

	// Wait for new event
	s.Eventually(func() bool {
		return len(s.received) >= 3
	}, time.Second, 10*time.Millisecond)

	// Should've received all 3 events
	s.Require().Lenf(s.received, 3, "received: %v", s.received)
	s.verifyMsg(s.received[0], 1)
	s.verifyMsg(s.received[1], 2)
	s.verifyMsg(s.received[2], 3)
}

func (s *ForwarderE2ETestSuite) TestNoResponders() {
	// Create forwarder
	s.startForwarder()

	// Stop subscriber
	s.Require().NoError(s.subscriber.Close(s.ctx))

	// Store events
	s.storeEvent(1)
	s.storeEvent(2)

	// sleep and verify no messages were received
	time.Sleep(100 * time.Millisecond)
	s.Require().Empty(s.received)

	// Restart subscriber
	s.Require().NoError(s.subscriber.Subscribe(s.ctx, "test"))

	// Store more events
	s.storeEvent(3)
	s.storeEvent(4)

	// Wait for event to be published
	s.Eventually(func() bool {
		return len(s.received) >= 2
	}, time.Second, 10*time.Millisecond)

	// Verify the messages
	s.Require().Lenf(s.received, 2, "received: %v", s.received)
	s.verifyMsg(s.received[0], 3)
	s.verifyMsg(s.received[1], 4)
}

func (s *ForwarderE2ETestSuite) TestRestart() {
	// Create forwarder
	f := s.startForwarder()

	// Store some events
	s.storeEvents(3)

	// Wait for events
	s.Eventually(func() bool {
		return len(s.received) == 3
	}, time.Second, 10*time.Millisecond)

	// Stop forwarder
	s.Require().NoError(f.Stop(s.ctx))
	s.received = s.received[:0]

	// Store more events while stopped
	s.storeEvent(4)
	s.storeEvent(5)

	// Start new forwarder - should process all events from beginning
	s.startForwarder()

	// Should receive all events since forwarder doesn't checkpoint
	s.Eventually(func() bool {
		return len(s.received) == 5
	}, time.Second, 10*time.Millisecond)

	for i, msg := range s.received {
		s.verifyMsg(msg, i+1)
	}
}

func (s *ForwarderE2ETestSuite) storeEvent(index int) {
	err := s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "test",
		Object:     fmt.Sprintf("event-%d", index),
	})
	s.Require().NoError(err)
}

func (s *ForwarderE2ETestSuite) storeEvents(count int) {
	for i := 1; i <= count; i++ {
		s.storeEvent(i)
	}
}

func (s *ForwarderE2ETestSuite) verifyMsg(msg *envelope.Message, i int) {
	payload, ok := msg.GetPayload("")
	s.Require().True(ok, "payload missing or not a string")
	s.Contains(payload, fmt.Sprintf("event-%d", i))
	s.Require().Equal(fmt.Sprintf("%d", i), msg.Metadata.Get(transport.KeySeqNum))
}

// Helper implementation
type testMessageCreator struct{}

func (c *testMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	return envelope.NewMessage(event.Object), nil
}

func TestForwarderE2ETestSuite(t *testing.T) {
	suite.Run(t, new(ForwarderE2ETestSuite))
}
