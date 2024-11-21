//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type OrderedPublisherTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	serializer *envelope.Serializer
	registry   *envelope.Registry
	publisher  OrderedPublisher
	ctx        context.Context
	cancel     context.CancelFunc
}

func (suite *OrderedPublisherTestSuite) SetupSuite() {
	suite.serializer = envelope.NewSerializer()
	suite.registry = envelope.NewRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *OrderedPublisherTestSuite) TearDownSuite() {
	if suite.natsConn != nil {
		suite.natsConn.Close()
	}
	if suite.natsServer != nil {
		suite.natsServer.Shutdown()
	}
}

func (suite *OrderedPublisherTestSuite) SetupTest() {
	var err error
	suite.ctx, suite.cancel = context.WithCancel(context.Background())

	suite.publisher, err = NewOrderedPublisher(suite.natsConn, OrderedPublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       TestSubject,
		AckWait:           500 * time.Millisecond,
		MaxPending:        10,
	})
	suite.Require().NoError(err)
}

func (suite *OrderedPublisherTestSuite) TearDownTest() {
	if suite.publisher != nil {
		suite.Require().NoError(suite.publisher.Close(suite.ctx))
	}
	if suite.cancel != nil {
		suite.cancel()
	}
}

func (suite *OrderedPublisherTestSuite) publishAndVerify(subject string, req PublishRequest) *envelope.Message {
	// Set up responder
	sub, err := suite.natsConn.Subscribe(subject, func(msg *nats.Msg) {
		suite.Require().NoError(Ack(msg))
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	// Verify the message is published successfully
	err = suite.publisher.Publish(suite.ctx, req)
	suite.Require().NoError(err)

	return req.Message
}

func (suite *OrderedPublisherTestSuite) TestBasicPublish() {
	event := TestPayload{Message: "Hello, World!"}
	suite.publishAndVerify(TestSubject, NewPublishRequest(envelope.NewMessage(event)))
}

func (suite *OrderedPublisherTestSuite) TestPublishWithMetadata() {
	event := TestPayload{Message: "Hello, World!"}
	metadata := &envelope.Metadata{"CustomKey": "CustomValue"}
	message := suite.publishAndVerify(TestSubject, NewPublishRequest(envelope.NewMessage(event).WithMetadata(metadata)))
	suite.Equal("CustomValue", message.Metadata.Get("CustomKey"))
}

func (suite *OrderedPublisherTestSuite) TestPublishWithDestinationPrefix() {
	var err error
	suite.publisher, err = NewOrderedPublisher(suite.natsConn, OrderedPublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		DestinationPrefix: TestDestinationPrefix,
		AckWait:           500 * time.Millisecond,
	})
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	subject := fmt.Sprintf("%s.%s", TestDestinationPrefix, TestPayloadType)
	suite.publishAndVerify(subject, NewPublishRequest(envelope.NewMessage(event)))
}

func (suite *OrderedPublisherTestSuite) TestTimeout() {
	sub, err := suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		// Don't respond - should trigger timeout
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	err = suite.publisher.Publish(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: "test"})))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "publish ack timeout")
}

func (suite *OrderedPublisherTestSuite) TestNoResponders() {
	// No responder, so publish should fail
	err := suite.publisher.Publish(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: "test"})))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "no responders available")
}

func (suite *OrderedPublisherTestSuite) TestMaxInflight() {
	var publishedMsgs []*nats.Msg
	sub, err := suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		// Set up a no-op responder
		publishedMsgs = append(publishedMsgs, msg)
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	// Create publisher with small queue
	pub, err := NewOrderedPublisher(suite.natsConn, OrderedPublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       TestSubject,
		MaxPending:        2,
		AckWait:           2 * time.Second,
	})
	suite.Require().NoError(err)
	defer pub.Close(suite.ctx)

	// First two should succeed
	for i := 1; i <= 2; i++ {
		_, err = pub.PublishAsync(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: fmt.Sprintf("test%d", i)})))
		suite.Require().NoError(err)
		time.Sleep(50 * time.Millisecond) // little backoff for publishLoop to process the msg
	}

	// Third should fail due to max inflight
	_, err = pub.PublishAsync(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: "test3"})))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "max pending messages reached")

	// ensure only 2 messages are published
	suite.Eventuallyf(func() bool {
		return len(publishedMsgs) == 2
	}, 1*time.Second, 50*time.Millisecond, "expected 2 messages to be published, got %d", len(publishedMsgs))
}

func (suite *OrderedPublisherTestSuite) TestReset() {
	futures := make([]PubFuture, 0)
	sub, err := suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		// no response to ensure messages are pending
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	// Queue up some messages
	for i := 0; i < 5; i++ {
		future, err := suite.publisher.PublishAsync(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{
			Message: fmt.Sprintf("test%d", i),
		})))
		suite.Require().NoError(err)
		futures = append(futures, future)
	}

	// Reset should cancel all pending
	suite.publisher.Reset(suite.ctx)

	for _, future := range futures {
		suite.Require().NoError(future.Wait(suite.ctx))
		suite.Require().Error(future.Err())
		suite.Require().Contains(future.Err().Error(), "publisher reset")
	}
	suite.T().Logf("Verified all futures")

	// Should be able to publish after reset
	sub.Unsubscribe()
	sub, err = suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		suite.Require().NoError(Ack(msg))
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	err = suite.publisher.Publish(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: "after-reset"})))
	suite.Require().NoError(err)
	suite.T().Log("Published after reset")
}

func (suite *OrderedPublisherTestSuite) TestConcurrentPublish() {
	const numGoroutines = 10
	const numMessages = 10

	sub, err := suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		suite.Require().NoError(Ack(msg))
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				err := suite.publisher.Publish(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{
					Message: fmt.Sprintf("msg-%d-%d", n, j),
				})))
				suite.Require().NoError(err)
			}
		}(i)
	}

	wg.Wait()
}

func (suite *OrderedPublisherTestSuite) TestNack() {
	sub, err := suite.natsConn.Subscribe(TestSubject, func(msg *nats.Msg) {
		suite.Require().NoError(Nack(msg, errors.New("test error")))
	})
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	err = suite.publisher.Publish(suite.ctx, NewPublishRequest(envelope.NewMessage(TestPayload{Message: "test-nack"})))
	suite.Require().Error(err)
	suite.Require().Contains(err.Error(), "test error")
}

func TestOrderedPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(OrderedPublisherTestSuite))
}
