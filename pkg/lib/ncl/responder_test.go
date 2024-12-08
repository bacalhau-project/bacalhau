//go:build unit || !integration

package ncl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
)

type ResponderTestSuite struct {
	suite.Suite
	natsServer        *server.Server
	natsConn          *nats.Conn
	serializer        *envelope.Serializer
	registry          *envelope.Registry
	responder         Responder
	processingTimeout time.Duration
}

func (suite *ResponderTestSuite) SetupSuite() {
	suite.serializer = envelope.NewSerializer()
	suite.registry = envelope.NewRegistry()
	suite.Require().NoError(suite.registry.Register(TestPayloadType, TestPayload{}))

	suite.natsServer, suite.natsConn = StartNats(suite.T())
}

func (suite *ResponderTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *ResponderTestSuite) SetupTest() {
	suite.processingTimeout = 100 * time.Millisecond
	var err error
	suite.responder, err = NewResponder(suite.natsConn, ResponderConfig{
		Name:              "test-responder",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
		Subject:           TestSubject,
		ProcessingTimeout: suite.processingTimeout,
	})
	suite.Require().NoError(err)
}

func (suite *ResponderTestSuite) TearDownTest() {
	if suite.responder != nil {
		suite.Require().NoError(suite.responder.Close(context.Background()))
	}
}

func (suite *ResponderTestSuite) makeRequest(msg *envelope.Message) (*envelope.Message, error) {
	// Create a publisher to make requests
	pub, err := NewPublisher(suite.natsConn, PublisherConfig{
		Name:              "test-publisher",
		MessageSerializer: suite.serializer,
		MessageRegistry:   suite.registry,
	})
	suite.Require().NoError(err)

	// Make request
	return pub.Request(context.Background(), NewPublishRequest(msg).WithSubject(TestSubject))
}

func (suite *ResponderTestSuite) TestBasicRequestResponse() {
	// Register handler
	err := suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			payload, ok := msg.GetPayload(TestPayload{})
			if !ok {
				return nil, errors.New("invalid payload")
			}
			return envelope.NewMessage(TestPayload{
				Message: "Response: " + payload.(TestPayload).Message,
			}), nil
		}))
	suite.Require().NoError(err)

	// Make request
	request := envelope.NewMessage(TestPayload{Message: "Hello"})
	response, err := suite.makeRequest(request)

	// Verify response
	suite.Require().NoError(err)
	suite.NotNil(response)
	payload, ok := response.GetPayload(TestPayload{})
	suite.True(ok)
	suite.Equal("Response: Hello", payload.(TestPayload).Message)
}

func (suite *ResponderTestSuite) TestHandlerTimeout() {
	// Register slow handler
	err := suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(2 * suite.processingTimeout):
				return envelope.NewMessage(TestPayload{Message: "Too Late"}), nil
			}
		}))
	suite.Require().NoError(err)

	request := envelope.NewMessage(TestPayload{Message: "Hello"})
	response, err := suite.makeRequest(request)

	suite.Require().Error(err)
	suite.Nil(response)
	suite.Contains(err.Error(), "context deadline exceeded")
}

func (suite *ResponderTestSuite) TestHandlerError() {
	// Register handler that returns error
	err := suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return nil, errors.New("handler error")
		}))
	suite.Require().NoError(err)

	// Make request
	request := envelope.NewMessage(TestPayload{Message: "Hello"})
	response, err := suite.makeRequest(request)

	suite.Require().Error(err)
	suite.Nil(response)
	suite.Contains(err.Error(), "handler error")
}

func (suite *ResponderTestSuite) TestNoHandlerForType() {
	// Register handler for different message type
	err := suite.responder.Listen(context.Background(), "TestAnotherType", RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return nil, nil
		}))
	suite.Require().NoError(err)

	// Make request without registering handler
	request := envelope.NewMessage(TestPayload{Message: "Hello"})
	response, err := suite.makeRequest(request)

	suite.Require().Error(err)
	suite.Nil(response)
	suite.Contains(err.Error(), "no handler found for message type")
}

func (suite *ResponderTestSuite) TestMultipleHandlers() {
	// Register first handler
	err := suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return envelope.NewMessage(TestPayload{Message: "First Handler"}), nil
		}))
	suite.Require().NoError(err)

	// Try to register second handler for same type
	err = suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return envelope.NewMessage(TestPayload{Message: "Second Handler"}), nil
		}))
	suite.Require().Error(err)
	suite.ErrorIs(err, ErrHandlerExists)
}

func (suite *ResponderTestSuite) TestInvalidMessages() {
	handler := RequestHandlerFunc(func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
		return envelope.NewMessage(TestPayload{Message: "Response"}), nil
	})
	suite.Require().NoError(suite.responder.Listen(context.Background(), TestPayloadType, handler))

	// Send invalid message bytes
	response := suite.natsConn.PublishRequest(TestSubject, "reply", []byte("invalid"))
	suite.Require().NoError(response)
}

func (suite *ResponderTestSuite) TestCloseAndReopen() {
	// Register handler
	err := suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return envelope.NewMessage(TestPayload{Message: "Response"}), nil
		}))
	suite.Require().NoError(err)

	// Close responder
	err = suite.responder.Close(context.Background())
	suite.Require().NoError(err)

	// Try request after close
	request := envelope.NewMessage(TestPayload{Message: "Hello"})
	response, err := suite.makeRequest(request)
	suite.Require().Error(err)
	suite.Nil(response)

	// Reopen responder
	suite.SetupTest()
	err = suite.responder.Listen(context.Background(), TestPayloadType, RequestHandlerFunc(
		func(ctx context.Context, msg *envelope.Message) (*envelope.Message, error) {
			return envelope.NewMessage(TestPayload{Message: "Response"}), nil
		}))
	suite.Require().NoError(err)

	// Request should work again
	response, err = suite.makeRequest(request)
	suite.Require().NoError(err)
	suite.NotNil(response)
}

func TestResponderTestSuite(t *testing.T) {
	suite.Run(t, new(ResponderTestSuite))
}
