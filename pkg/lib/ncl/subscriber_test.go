//go:build unit || !integration

package ncl

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

type SubscriberTestSuite struct {
	suite.Suite
	natsServer     *server.Server
	natsConn       *nats.Conn
	serializer     *EnvelopeSerializer
	registry       *PayloadRegistry
	publisher      Publisher
	subscriber     Subscriber
	messageHandler *TestMessageHandler
}

func (suite *SubscriberTestSuite) SetupSuite() {
	suite.serializer = NewEnvelopeSerializer()
	suite.registry = NewPayloadRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *SubscriberTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *SubscriberTestSuite) SetupTest() {
	var err error
	suite.publisher, err = NewPublisher(
		suite.natsConn,
		WithPublisherName("test"),
		WithPublisherDestination(TestSubject),
		WithPublisherMessageSerializer(suite.serializer),
		WithPublisherPayloadRegistry(suite.registry),
	)
	suite.Require().NoError(err)

	suite.messageHandler = &TestMessageHandler{}
	suite.subscriber, err = NewSubscriber(
		suite.natsConn,
		WithSubscriberMessageHandlers(suite.messageHandler),
		WithSubscriberMessageDeserializer(suite.serializer),
		WithSubscriberPayloadRegistry(suite.registry),
	)
	suite.Require().NoError(err)
}

func (suite *SubscriberTestSuite) TearDownTest() {
	err := suite.subscriber.Close(context.Background())
	suite.Require().NoError(err)
}

func (suite *SubscriberTestSuite) TestSubscribe() {
	err := suite.subscriber.Subscribe(TestSubject)
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	err = suite.publisher.Publish(context.Background(), event)
	suite.Require().NoError(err)

	// Wait for message to be processed
	suite.Eventually(func() bool {
		return len(suite.messageHandler.messages) > 0
	}, 1000*time.Millisecond, 50*time.Millisecond, "Message not processed")

	suite.Require().Len(suite.messageHandler.messages, 1)
	suite.Equal("Hello, World!", suite.messageHandler.messages[0].Payload.(*TestPayload).Message)
}

func (suite *SubscriberTestSuite) TestSubscribeWithFilter() {
	filter := func(metadata *Metadata) bool {
		return metadata.Get("filter") == "true"
	}

	var err error
	suite.subscriber, err = NewSubscriber(
		suite.natsConn,
		WithSubscriberMessageHandlers(suite.messageHandler),
		WithSubscriberMessageDeserializer(suite.serializer),
		WithSubscriberPayloadRegistry(suite.registry),
		WithSubscriberMessageFilter(MessageFilterFunc(filter)),
	)
	suite.Require().NoError(err)

	err = suite.subscriber.Subscribe(TestSubject)
	suite.Require().NoError(err)

	// Publish a message that should be filtered out
	event1 := TestPayload{Message: "Filtered"}
	err = suite.publisher.PublishWithMetadata(context.Background(), &Metadata{"filter": "true"}, event1)
	suite.Require().NoError(err)

	// Publish a message that should be processed
	event2 := TestPayload{Message: "Not Filtered"}
	err = suite.publisher.PublishWithMetadata(context.Background(), &Metadata{"filter": "false"}, event2)
	suite.Require().NoError(err)

	// Wait for message to be processed
	suite.Eventually(func() bool {
		return len(suite.messageHandler.messages) > 0
	}, 1000*time.Millisecond, 50*time.Millisecond, "Message not processed")

	suite.Require().Len(suite.messageHandler.messages, 1)
	suite.Equal("Not Filtered", suite.messageHandler.messages[0].Payload.(*TestPayload).Message)
}

func (suite *SubscriberTestSuite) TestMultipleSubscriptions() {
	err := suite.subscriber.Subscribe(TestSubject, TestDestinationPrefix+".>")
	suite.Require().NoError(err)

	event1 := TestPayload{Message: "Message 1"}
	err = suite.publisher.Publish(context.Background(), event1)
	suite.Require().NoError(err)

	publisher2, err := NewPublisher(
		suite.natsConn,
		WithPublisherName("test"),
		WithPublisherDestinationPrefix(TestDestinationPrefix),
		WithPublisherMessageSerializer(suite.serializer),
		WithPublisherPayloadRegistry(suite.registry),
	)
	suite.Require().NoError(err)

	event2 := TestPayload{Message: "Message 2"}
	err = publisher2.Publish(context.Background(), event2)
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
	err := suite.subscriber.Subscribe(TestSubject)
	suite.Require().NoError(err)

	err = suite.subscriber.Close(context.Background())
	suite.Require().NoError(err)

	// Try to publish after closing
	event := TestPayload{Message: "Hello, World!"}
	err = suite.publisher.Publish(context.Background(), event)
	suite.Require().NoError(err)

	// Wait for a short time
	time.Sleep(200 * time.Millisecond)
	suite.Require().Len(suite.messageHandler.messages, 0)
}

func TestSubscriberTestSuite(t *testing.T) {
	suite.Run(t, new(SubscriberTestSuite))
}
