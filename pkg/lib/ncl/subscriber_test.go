//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type SubscriberTestSuite struct {
	suite.Suite
	natsServer       *server.Server
	natsConn         *nats.Conn
	serializer       *envelope.Serializer
	registry         *envelope.Registry
	publisher        Publisher
	orderedPublisher *orderedPublisher
	subscriber       Subscriber
	messageHandler   *TestMessageHandler
	notifier         *TestNotifier
	acks             []*nats.Msg
}

func (suite *SubscriberTestSuite) SetupSuite() {
	suite.serializer = envelope.NewSerializer()
	suite.registry = envelope.NewRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *SubscriberTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *SubscriberTestSuite) SetupTest() {
	var err error
	suite.publisher, err = NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       TestSubject,
	})
	suite.Require().NoError(err)

	pub, err := NewOrderedPublisher(suite.natsConn, OrderedPublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       TestSubject,
	})
	suite.Require().NoError(err)
	suite.orderedPublisher = pub.(*orderedPublisher)

	// subscribe to acks for ordered publisher
	var ackMutex sync.Mutex
	subscription, err := suite.natsConn.Subscribe(suite.orderedPublisher.inbox+".*", func(msg *nats.Msg) {
		ackMutex.Lock()
		defer ackMutex.Unlock()
		suite.T().Log("Received ack", string(msg.Data))
		suite.acks = append(suite.acks, msg)

	})
	suite.Require().NoError(err)
	suite.T().Cleanup(func() {
		suite.Require().NoError(subscription.Unsubscribe())
	})

	suite.messageHandler = &TestMessageHandler{}
	suite.notifier = &TestNotifier{}
	suite.subscriber, err = NewSubscriber(suite.natsConn, SubscriberConfig{
		Name:               "test",
		MessageSerializer:  suite.serializer,
		MessageRegistry:    suite.registry,
		MessageHandler:     suite.messageHandler,
		ProcessingNotifier: suite.notifier,
	})
	suite.Require().NoError(err)
}

func (suite *SubscriberTestSuite) TearDownTest() {
	err := suite.subscriber.Close(context.Background())
	suite.Require().NoError(err)
}

func (suite *SubscriberTestSuite) TestSubscribe() {
	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(envelope.NewMessage(event)))
	suite.Require().NoError(err)

	// Wait for message to be processed
	suite.Eventually(func() bool {
		return len(suite.messageHandler.messages) > 0
	}, 1000*time.Millisecond, 50*time.Millisecond, "Message not processed")

	suite.Require().Len(suite.messageHandler.messages, 1)
	suite.Equal("Hello, World!", suite.messageHandler.messages[0].Payload.(*TestPayload).Message)

	// Verify notifier
	suite.Require().Len(suite.notifier.notifications, 1)
	suite.Equal("Hello, World!", suite.notifier.notifications[0].Payload.(*TestPayload).Message)
}

func (suite *SubscriberTestSuite) TestSubscribeWithFilter() {
	filter := func(metadata nats.Header) bool {
		return metadata.Get("filter") == "true"
	}

	var err error
	suite.subscriber, err = NewSubscriber(suite.natsConn, SubscriberConfig{
		Name:               "test",
		MessageSerializer:  suite.serializer,
		MessageRegistry:    suite.registry,
		MessageHandler:     suite.messageHandler,
		MessageFilter:      MessageFilterFunc(filter),
		ProcessingNotifier: suite.notifier,
	})
	suite.Require().NoError(err)

	err = suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	// Publish a message that should be filtered out
	event1 := TestPayload{Message: "Filtered"}
	msg1 := envelope.NewMessage(event1).WithMetadataValue("filter", "true")
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(msg1))
	suite.Require().NoError(err)

	// Publish a message that should be processed
	event2 := TestPayload{Message: "Not Filtered"}
	msg2 := envelope.NewMessage(event2).WithMetadataValue("filter", "false")
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(msg2))
	suite.Require().NoError(err)

	// Wait for message to be processed
	suite.Eventually(func() bool {
		return len(suite.messageHandler.messages) > 0
	}, 1000*time.Millisecond, 50*time.Millisecond, "Message not processed")

	suite.Require().Len(suite.messageHandler.messages, 1)
	suite.Equal("Not Filtered", suite.messageHandler.messages[0].Payload.(*TestPayload).Message)

	// Verify notifier
	suite.Require().Len(suite.notifier.notifications, 1)
	suite.Equal("Not Filtered", suite.notifier.notifications[0].Payload.(*TestPayload).Message)
}

func (suite *SubscriberTestSuite) TestSubscribeErrorHandling() {
	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	// Set handler to fail
	suite.messageHandler.WithFailureMessage("handler error")

	// Publish a message
	event := TestPayload{Message: "Will fail"}
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(envelope.NewMessage(event)))
	suite.Require().NoError(err)

	// Wait a bit to ensure no notification is sent
	time.Sleep(100 * time.Millisecond)

	suite.Len(suite.notifier.notifications, 0, "Should not receive notification when handler fails")
}

func (suite *SubscriberTestSuite) TestSubscribeWithAck() {
	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)
	suite.publishAndVerifyAck(nil)
}

func (suite *SubscriberTestSuite) TestSubscribeWithNack() {
	// mock message handler to return an error
	suite.messageHandler.WithFailureMessage("my error")

	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	suite.publishAndVerifyAck(errors.New("my error"))
}

func (suite *SubscriberTestSuite) TestSubscribeWithNackDelays() {
	// mock message handler to return an error
	suite.messageHandler.WithFailureMessage("my error")

	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	// First failure should have initial delay
	nack1 := suite.publishAndVerifyAck(errors.New("my error"))
	suite.Require().Greater(nack1.Delay, time.Duration(0))

	// Second failure should have longer delay
	nack2 := suite.publishAndVerifyAck(errors.New("my error"))
	suite.Require().Greater(nack2.Delay, nack1.Delay)

	// Third failure should have even longer delay
	nack3 := suite.publishAndVerifyAck(errors.New("my error"))
	suite.Require().Greater(nack3.Delay, nack2.Delay)

	// Now let's succeed to reset the backoff
	suite.messageHandler.WithFailureMessage("")
	suite.publishAndVerifyAck(nil)

	// Now fail again - should be back to initial delay
	suite.messageHandler.WithFailureMessage("my error")
	nack4 := suite.publishAndVerifyAck(errors.New("my error"))
	suite.Require().Equal(nack4.Delay, nack1.Delay)
}

func (suite *SubscriberTestSuite) TestMultipleSubscriptions() {
	err := suite.subscriber.Subscribe(context.Background(), TestSubject, TestDestinationPrefix+".>")
	suite.Require().NoError(err)

	event1 := TestPayload{Message: "Message 1"}
	msg1 := envelope.NewMessage(event1)
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(msg1))
	suite.Require().NoError(err)

	publisher2, err := NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		DestinationPrefix: TestDestinationPrefix,
	})
	suite.Require().NoError(err)

	event2 := TestPayload{Message: "Message 2"}
	msg2 := envelope.NewMessage(event2)
	err = publisher2.Publish(context.Background(), NewPublishRequest(msg2))
	suite.Require().NoError(err)

	// Wait for messages to be processed
	suite.Eventually(func() bool {
		return len(suite.messageHandler.messages) >= 2
	}, 1000*time.Millisecond, 50*time.Millisecond, "Expected 2 messages, but got %d", len(suite.messageHandler.messages))

	suite.Require().Len(suite.messageHandler.messages, 2)

	// Check that both messages are received, regardless of order
	receivedMessages := make(map[string]bool)
	for _, msg := range suite.messageHandler.messages {
		payload, ok := msg.Payload.(*TestPayload)
		suite.Require().True(ok, "Expected payload of type *TestPayload")
		receivedMessages[payload.Message] = true
	}

	suite.Require().True(receivedMessages["Message 1"], "Message 1 not received")
	suite.Require().True(receivedMessages["Message 2"], "Message 2 not received")
}

func (suite *SubscriberTestSuite) TestClose() {
	err := suite.subscriber.Subscribe(context.Background(), TestSubject)
	suite.Require().NoError(err)

	err = suite.subscriber.Close(context.Background())
	suite.Require().NoError(err)

	// Try to publish after closing
	event := TestPayload{Message: "Hello, World!"}
	err = suite.publisher.Publish(context.Background(), NewPublishRequest(envelope.NewMessage(event)))
	suite.Require().NoError(err)

	// Wait for a short time
	time.Sleep(200 * time.Millisecond)
	suite.Require().Len(suite.messageHandler.messages, 0)
}

func (suite *SubscriberTestSuite) publishAndVerifyAck(nackErr error) *Result {
	// reset ack msgs first
	suite.acks = suite.acks[:0]

	// Publish message
	event := TestPayload{Message: "Message"}
	err := suite.orderedPublisher.Publish(context.Background(), NewPublishRequest(envelope.NewMessage(event)))

	// Verify error, if any
	if nackErr != nil {
		suite.Require().Error(err)
		suite.Require().Contains(err.Error(), nackErr.Error())
	} else {
		suite.Require().NoError(err)
	}

	// wait for ack/nack
	suite.Eventuallyf(func() bool {
		return len(suite.acks) > 0
	}, 1000*time.Millisecond, 50*time.Millisecond, "Expected ack/nack")

	// Verify ack/nack
	suite.Require().Len(suite.acks, 1)
	res, err := ParseResult(suite.acks[len(suite.acks)-1])
	suite.Require().NoError(err)

	if nackErr != nil {
		suite.Require().Error(res.Err())
		suite.Require().Contains(res.Err().Error(), nackErr.Error())
	} else {
		suite.Require().NoError(res.Err())
		suite.Require().Zero(res.Delay)
	}
	return res
}

func TestSubscriberTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriberTestSuite))
}
