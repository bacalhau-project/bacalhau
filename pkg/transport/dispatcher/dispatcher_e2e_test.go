//go:build unit || !integration

package dispatcher_test

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
	"github.com/bacalhau-project/bacalhau/pkg/transport/dispatcher"
)

type DispatcherE2ETestSuite struct {
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

func (s *DispatcherE2ETestSuite) SetupTest() {
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
	var msgHandler ncl.MessageHandlerFunc
	msgHandler = func(_ context.Context, msg *envelope.Message) error {
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

func (s *DispatcherE2ETestSuite) TearDownTest() {
	// Run cleanup functions in LIFO order
	for i := len(s.cleanupFuncs) - 1; i >= 0; i-- {
		s.cleanupFuncs[i]()
	}
	s.cleanupFuncs = nil // Clear the slice

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

// startDispatcher creates a new dispatcher, along with watcher nand publisher with default configuration
func (s *DispatcherE2ETestSuite) startDispatcher(config dispatcher.Config) *dispatcher.Dispatcher {
	w, err := watcher.New(s.ctx, "test-watcher", s.store)
	s.Require().NoError(err)

	// Create publisher
	publisher, err := ncl.NewOrderedPublisher(s.nc, ncl.OrderedPublisherConfig{
		Name:              "test-publisher",
		Destination:       "test",
		MessageRegistry:   s.registry,
		MessageSerializer: envelope.NewSerializer(),
	})
	s.Require().NoError(err)

	d, err := dispatcher.New(publisher, w, &testMessageCreator{}, config)
	s.Require().NoError(err)

	s.Require().NoError(d.Start(s.ctx))

	s.cleanupFuncs = append(s.cleanupFuncs, func() {
		s.Require().NoError(d.Stop(s.ctx))
		s.Require().NoError(publisher.Close(s.ctx))
	})

	return d
}

func (s *DispatcherE2ETestSuite) TestEventFlow() {
	// Create dispatcher
	s.startDispatcher(dispatcher.DefaultConfig())

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

func (s *DispatcherE2ETestSuite) TestRecovery() {
	// Create dispatcher with fast retry intervals for testing
	s.startDispatcher(dispatcher.Config{
		BaseRetryInterval:  100 * time.Millisecond,
		MaxRetryInterval:   100 * time.Millisecond,
		CheckpointInterval: time.Hour, // disable checkpointing
	})

	// Store an event
	s.storeEvent(1)

	// Wait for event to be published
	s.Eventually(func() bool {
		return len(s.received) == 1
	}, 2*time.Second, 10*time.Millisecond)

	// Verify received message
	s.Require().Lenf(s.received, 1, "received: %v", s.received)
	s.verifyMsg(s.received[0], 1)

	// Clear received messages
	s.received = s.received[:0]

	// Stop NATS server to fail publishing
	s.natsServer.Shutdown()
	s.natsServer.WaitForShutdown()

	// wait for client to be disconnected
	s.Eventually(func() bool {
		return !s.nc.IsConnected()
	}, time.Second, 10*time.Millisecond)
	s.T().Logf("NATS server stopped. Client status: %s", s.nc.Status())

	// Store another event
	s.storeEvent(2)

	// no new events should be received
	time.Sleep(500 * time.Millisecond)
	s.Require().Emptyf(s.received, "received: %v", s.received)

	// Restart NATS server
	s.natsServer, _ = testutils.RestartNatsServer(s.T(), s.natsServer)
	s.Require().Eventually(func() bool {
		return s.nc.IsConnected()
	}, 5*time.Second, 10*time.Millisecond)
	s.T().Logf("NATS server restarted. Client status: %s", s.nc.Status())

	// Wait for dispatcher to recover
	// We should receive two events as the first event should be retried because we didn't checkpoint
	s.Eventually(func() bool {
		return len(s.received) == 2
	}, time.Second, 10*time.Millisecond)

	// Verify received messages
	s.Require().Lenf(s.received, 2, "received: %v", s.received)
	s.verifyMsg(s.received[0], 1)
	s.verifyMsg(s.received[1], 2)
}

func (s *DispatcherE2ETestSuite) TestCheckpointingAndRestart() {
	// Create dispatcher with frequent checkpointing
	config := dispatcher.Config{
		CheckpointInterval: 100 * time.Millisecond,
	}
	d := s.startDispatcher(config)

	// Store some events
	s.storeEvents(5)

	// Wait for events to be processed and checkpointed
	s.Eventually(func() bool {
		checkpoint, err := s.store.GetCheckpoint(s.ctx, "test-watcher")
		if err != nil {
			return false
		}
		return checkpoint == 5
	}, time.Second, 10*time.Millisecond)

	// Stop dispatcher
	s.Require().NoError(d.Stop(s.ctx))
	s.received = s.received[:0]

	// Store more events while stopped
	s.storeEvent(6)
	s.storeEvent(7)
	s.storeEvent(8)

	// Start new dispatcher - should resume from checkpoint
	s.startDispatcher(config)

	// Should only receive events after checkpoint
	s.Eventually(func() bool {
		return len(s.received) == 3
	}, time.Second, 10*time.Millisecond)

	for i, msg := range s.received {
		s.verifyMsg(msg, i+6)
	}
}

func (s *DispatcherE2ETestSuite) storeEvent(index int) {
	err := s.store.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation:  watcher.OperationCreate,
		ObjectType: "test",
		Object:     fmt.Sprintf("event-%d", index),
	})
	s.Require().NoError(err)
}

func (s *DispatcherE2ETestSuite) storeEvents(count int) {
	for i := 1; i <= count; i++ {
		s.storeEvent(i)
	}
}

func (s *DispatcherE2ETestSuite) verifyMsg(msg *envelope.Message, i int) {
	payload, ok := msg.GetPayload("")
	s.Require().True(ok, "payload missing or not a string")
	s.Contains(payload, fmt.Sprintf("event-%d", i))
	s.Require().Equal(fmt.Sprintf("%d", i), msg.Metadata.Get(dispatcher.KeySeqNum))
}

// Helper implementation
type testMessageCreator struct{}

func (c *testMessageCreator) CreateMessage(event watcher.Event) (*envelope.Message, error) {
	return envelope.NewMessage(event.Object), nil
}

func TestDispatcherE2ETestSuite(t *testing.T) {
	suite.Run(t, new(DispatcherE2ETestSuite))
}
