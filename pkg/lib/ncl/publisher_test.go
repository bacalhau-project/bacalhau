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
	serializer *EnvelopedRawMessageSerDe
	registry   *MessageSerDeRegistry
	publisher  Publisher
}

func (suite *PublisherTestSuite) SetupSuite() {
	suite.serializer = NewEnvelopedRawMessageSerDe()
	suite.registry = NewMessageSerDeRegistry()
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
		WithPublisherMessageSerDeRegistry(suite.registry),
	)
	suite.Require().NoError(err)
}

func (suite *PublisherTestSuite) publishAndVerify(subject string, msg *Message) *Message {
	sub, err := suite.natsConn.SubscribeSync(subject)
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	err = suite.publisher.Publish(context.Background(), msg)
	suite.Require().NoError(err)

	nMsg, err := sub.NextMsg(time.Second)
	suite.Require().NoError(err)

	rawMsg, err := suite.serializer.Deserialize(nMsg.Data)
	suite.Require().NoError(err)

	readMsg, err := suite.registry.Deserialize(rawMsg)
	suite.Require().NoError(err)

	suite.Equal("test", readMsg.Metadata.Get(KeySource))
	suite.Equal(TestPayloadType, readMsg.Metadata.Get(KeyMessageType))
	suite.True(readMsg.Metadata.Has(KeyEventTime))
	suite.True(readMsg.IsType(TestPayload{}))

	readPayload, ok := readMsg.GetPayload(TestPayload{})
	suite.True(ok, "payload type not matched")
	suite.Equal(msg.Payload, readPayload)

	return readMsg
}

func (suite *PublisherTestSuite) TestPublish() {
	event := TestPayload{Message: "Hello, World!"}
	suite.publishAndVerify(TestSubject, NewMessage(event))
}

func (suite *PublisherTestSuite) TestPublishWithMetadata() {
	event := TestPayload{Message: "Hello, World!"}
	metadata := &Metadata{"CustomKey": "CustomValue"}
	message := suite.publishAndVerify(TestSubject, NewMessage(event).WithMetadata(metadata))
	suite.Equal("CustomValue", message.Metadata.Get("CustomKey"))
}

func (suite *PublisherTestSuite) TestPublishWithDestinationPrefix() {
	var err error
	suite.publisher, err = NewPublisher(
		suite.natsConn,
		WithPublisherName("test"),
		WithPublisherDestinationPrefix(TestDestinationPrefix),
		WithPublisherMessageSerializer(suite.serializer),
		WithPublisherMessageSerDeRegistry(suite.registry),
	)
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	subject := fmt.Sprintf("%s.%s", TestDestinationPrefix, TestPayloadType)
	suite.publishAndVerify(subject, NewMessage(event))
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
				payload := TestPayload{Message: fmt.Sprintf("Message %d", j)}
				message := NewMessage(payload)
				err := suite.publisher.Publish(context.Background(), message)
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
	message := NewMessage(TestPayload{Message: largeMessage})
	suite.Require().Error(suite.publisher.Publish(context.Background(), message), "expected error publishing large message")
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
