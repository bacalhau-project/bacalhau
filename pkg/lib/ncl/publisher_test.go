//go:build unit || !integration

package ncl

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"
)

type PublisherTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	serializer *EnvelopeSerializer
	registry   *PayloadRegistry
	publisher  Publisher
}

func (suite *PublisherTestSuite) SetupSuite() {
	suite.serializer = NewEnvelopeSerializer()
	suite.registry = NewPayloadRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *PublisherTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *PublisherTestSuite) SetupTest() {
	var err error
	suite.publisher, err = NewPublisher(
		suite.natsConn,
		WithPublisherName("test"),
		WithPublisherDestination(TestSubject),
		WithPublisherMessageSerializer(suite.serializer),
		WithPublisherPayloadRegistry(suite.registry),
	)
	suite.Require().NoError(err)
}

func (suite *PublisherTestSuite) publishAndVerify(subject string, event TestPayload, metadata *Metadata) *Message {
	sub, err := suite.natsConn.SubscribeSync(subject)
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	if metadata != nil {
		err = suite.publisher.PublishWithMetadata(context.Background(), metadata, event)
	} else {
		err = suite.publisher.Publish(context.Background(), event)
	}
	suite.Require().NoError(err)

	msg, err := sub.NextMsg(time.Second)
	suite.Require().NoError(err)

	rawMsg := &RawMessage{}
	err = suite.serializer.Deserialize(msg.Data, rawMsg)
	suite.Require().NoError(err)

	payload, err := suite.registry.DeserializePayload(rawMsg.Metadata, rawMsg.Payload)
	suite.Require().NoError(err)

	message := &Message{Metadata: rawMsg.Metadata, Payload: payload}

	suite.Equal("test", message.Metadata.Get(KeySource))
	suite.Equal(TestPayloadType, message.Metadata.Get(KeyMessageType))
	suite.True(message.Metadata.Has(KeyEventTime))
	suite.True(message.IsType(TestPayload{}))
	suite.Equal(event.Message, payload.(*TestPayload).Message)

	return message
}

func (suite *PublisherTestSuite) TestPublish() {
	event := TestPayload{Message: "Hello, World!"}
	suite.publishAndVerify(TestSubject, event, nil)
}

func (suite *PublisherTestSuite) TestPublishWithMetadata() {
	event := TestPayload{Message: "Hello, World!"}
	metadata := &Metadata{"CustomKey": "CustomValue"}
	message := suite.publishAndVerify(TestSubject, event, metadata)
	suite.Equal("CustomValue", message.Metadata.Get("CustomKey"))
}

func (suite *PublisherTestSuite) TestPublishWithDestinationPrefix() {
	var err error
	suite.publisher, err = NewPublisher(
		suite.natsConn,
		WithPublisherName("test"),
		WithPublisherDestinationPrefix(TestDestinationPrefix),
		WithPublisherMessageSerializer(suite.serializer),
		WithPublisherPayloadRegistry(suite.registry),
	)
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	subject := fmt.Sprintf("%s.%s", TestDestinationPrefix, TestPayloadType)
	suite.publishAndVerify(subject, event, nil)
}

func (suite *PublisherTestSuite) TestConcurrentPublish() {
	const numGoroutines = 10
	const numMessages = 100

	sub, err := suite.natsConn.SubscribeSync(TestSubject)
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	var wg sync.WaitGroup
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				event := TestPayload{Message: fmt.Sprintf("Message %d", j)}
				err := suite.publisher.Publish(context.Background(), event)
				suite.Require().NoError(err)
			}
		}()
	}

	wg.Wait()

	// Check that all messages were received
	for i := 0; i < numGoroutines*numMessages; i++ {
		_, err := sub.NextMsg(time.Second)
		suite.Require().NoError(err)
	}
}

func (suite *PublisherTestSuite) TestLargePayload() {
	largeMessage := strings.Repeat("a", 1024*1024) // 1MB message
	event := TestPayload{Message: largeMessage}
	suite.Require().Error(suite.publisher.Publish(context.Background(), event), "expected error publishing large message")
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
