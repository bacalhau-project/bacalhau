//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type PublisherTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	serializer *envelope.Serializer
	registry   *envelope.Registry
	publisher  Publisher
}

func (suite *PublisherTestSuite) SetupSuite() {
	suite.serializer = envelope.NewSerializer()
	suite.registry = envelope.NewRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *PublisherTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *PublisherTestSuite) SetupTest() {
	var err error
	suite.publisher, err = NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Destination:       TestSubject,
	})
	suite.Require().NoError(err)
}

func (suite *PublisherTestSuite) publishAndVerify(subject string, req PublishRequest) *envelope.Message {
	sub, err := suite.natsConn.SubscribeSync(subject)
	suite.Require().NoError(err)
	defer sub.Unsubscribe()

	err = suite.publisher.Publish(context.Background(), req)
	suite.Require().NoError(err)

	nMsg, err := sub.NextMsg(time.Second)
	suite.Require().NoError(err)

	rawMsg, err := suite.serializer.Deserialize(nMsg.Data)
	suite.Require().NoError(err)

	readMsg, err := suite.registry.Deserialize(rawMsg)
	suite.Require().NoError(err)

	suite.Equal("test", readMsg.Metadata.Get(KeySource))
	suite.Equal(TestPayloadType, readMsg.Metadata.Get(envelope.KeyMessageType))
	suite.True(readMsg.Metadata.Has(KeyEventTime))
	suite.True(readMsg.IsType(TestPayload{}))

	readPayload, ok := readMsg.GetPayload(TestPayload{})
	suite.True(ok, "payload type not matched")
	suite.Equal(req.Message.Payload, readPayload)

	return readMsg
}

func (suite *PublisherTestSuite) requestAndVerify(ctx context.Context, request PublishRequest, handler func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error)) (*envelope.Message, error) {
	// Setup responder
	responder, err := NewResponder(suite.natsConn, ResponderConfig{
		Name:              "test-responder",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Subject:           TestSubject,
	})
	suite.Require().NoError(err)
	defer responder.Close(context.Background())

	// Add handler for our test payload type
	err = responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(handler))
	suite.Require().NoError(err)

	// Make the request
	return suite.publisher.Request(ctx, request)
}

func (suite *PublisherTestSuite) TestPublish() {
	event := TestPayload{Message: "Hello, World!"}
	suite.publishAndVerify(TestSubject, NewPublishRequest(envelope.NewMessage(event)))
}

func (suite *PublisherTestSuite) TestPublishWithMetadata() {
	event := TestPayload{Message: "Hello, World!"}
	metadata := &envelope.Metadata{"CustomKey": "CustomValue"}
	message := suite.publishAndVerify(TestSubject, NewPublishRequest(envelope.NewMessage(event).WithMetadata(metadata)))
	suite.Equal("CustomValue", message.Metadata.Get("CustomKey"))
}

func (suite *PublisherTestSuite) TestPublishWithDestinationPrefix() {
	var err error
	suite.publisher, err = NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		DestinationPrefix: TestDestinationPrefix,
	})
	suite.Require().NoError(err)

	event := TestPayload{Message: "Hello, World!"}
	subject := fmt.Sprintf("%s.%s", TestDestinationPrefix, TestPayloadType)
	suite.publishAndVerify(subject, NewPublishRequest(envelope.NewMessage(event)))
}

func (suite *PublisherTestSuite) TestPublishWithSubject() {
	event := TestPayload{Message: "Hello, World!"}
	customSubject := "custom.subject"
	suite.publishAndVerify(customSubject, NewPublishRequest(envelope.NewMessage(event)).WithSubject(customSubject))
}

func (suite *PublisherTestSuite) TestPublishWithSubjectPrefix() {
	event := TestPayload{Message: "Hello, World!"}
	customSubjectPrefix := "custom.prefix"
	expectedSubject := fmt.Sprintf("%s.%s", customSubjectPrefix, TestPayloadType)
	suite.publishAndVerify(expectedSubject, NewPublishRequest(envelope.NewMessage(event)).WithSubjectPrefix(customSubjectPrefix))
}

func (suite *PublisherTestSuite) TestPublishValidation() {
	// Test publishing nil message
	err := suite.publisher.Publish(context.Background(), PublishRequest{})
	suite.Require().Error(err)
	suite.Contains(err.Error(), "cannot publish nil message")

	// Test publishing with both subject and subject prefix
	err = suite.publisher.Publish(context.Background(), PublishRequest{
		Message:       envelope.NewMessage(TestPayload{Message: "Test"}),
		Subject:       "test.subject",
		SubjectPrefix: "test.prefix",
	})
	suite.Require().Error(err)
	suite.Contains(err.Error(), "cannot specify both subject and subject prefix")

	// Test publishing without subject or subject prefix when destination and destination prefix are not set
	pub, err := NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
	})
	suite.Require().NoError(err)
	err = pub.Publish(context.Background(), PublishRequest{
		Message: envelope.NewMessage(TestPayload{Message: "Test"}),
	})
	suite.Require().Error(err)
	suite.Contains(err.Error(), "must specify either subject or subject prefix")
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
				message := envelope.NewMessage(payload)
				err := suite.publisher.Publish(context.Background(), PublishRequest{Message: message})
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
	message := envelope.NewMessage(TestPayload{Message: largeMessage})
	suite.Require().Error(suite.publisher.Publish(context.Background(), PublishRequest{Message: message}), "expected error publishing large message")
}

func (suite *PublisherTestSuite) TestRequestSuccess() {
	handler := func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
		payload, ok := msg.GetPayload(TestPayload{})
		if !ok {
			return nil, fmt.Errorf("unexpected payload type: %T", msg.Payload)
		}

		return envelope.NewMessage(TestPayload{
			Message: "Response: " + payload.(TestPayload).Message,
		}), nil
	}

	request := NewPublishRequest(envelope.NewMessage(TestPayload{Message: "Hello"}))
	response, err := suite.requestAndVerify(context.Background(), request, handler)
	suite.Require().NoError(err)
	suite.Require().NotNil(response)

	// Verify response
	suite.Equal(TestPayloadType, response.Metadata.Get(envelope.KeyMessageType))
	payload, ok := response.GetPayload(TestPayload{})
	suite.True(ok)
	suite.Equal("Response: Hello", payload.(TestPayload).Message)
}

func (suite *PublisherTestSuite) TestRequestError() {
	handler := func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
		return nil, fmt.Errorf("test error")
	}

	request := NewPublishRequest(envelope.NewMessage(TestPayload{Message: "Hello"}))
	response, err := suite.requestAndVerify(context.Background(), request, handler)

	suite.Require().Error(err)
	suite.Nil(response)
	suite.Contains(err.Error(), "test error")
}

func (suite *PublisherTestSuite) TestRequestSlowResponse() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	handler := func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
		time.Sleep(200 * time.Millisecond)
		return envelope.NewMessage(TestPayload{Message: "too late"}), nil
	}

	request := NewPublishRequest(envelope.NewMessage(TestPayload{Message: "Hello"}))
	response, err := suite.requestAndVerify(ctx, request, handler)

	suite.Require().Error(err)
	suite.Nil(response)
	suite.True(errors.Is(err, context.DeadlineExceeded))
}

func (suite *PublisherTestSuite) TestRequestNoResponders() {
	// Send request without any subscribers
	req := NewPublishRequest(envelope.NewMessage(TestPayload{Message: "Hello"}))
	response, err := suite.publisher.Request(context.Background(), req)
	suite.Require().Error(err)
	suite.True(errors.Is(err, nats.ErrNoResponders))
	suite.Nil(response)
}

func (suite *PublisherTestSuite) TestRequestValidation() {
	testCases := []struct {
		name    string
		request PublishRequest
		errMsg  string
	}{
		{
			name:    "nil message",
			request: PublishRequest{},
			errMsg:  "cannot publish nil message",
		},
		{
			name: "both subject and prefix",
			request: PublishRequest{
				Message:       envelope.NewMessage(TestPayload{Message: "test"}),
				Subject:       "test.subject",
				SubjectPrefix: "test.prefix",
			},
			errMsg: "cannot specify both subject and subject prefix",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			response, err := suite.publisher.Request(context.Background(), tc.request)
			suite.Require().Error(err)
			suite.Contains(err.Error(), tc.errMsg)
			suite.Nil(response)
		})
	}
}

func TestPublisherTestSuite(t *testing.T) {
	suite.Run(t, new(PublisherTestSuite))
}
