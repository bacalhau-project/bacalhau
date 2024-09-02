//go:build unit || !integration

package compute

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type MessageForwarderTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	publisher  ncl.Publisher
	forwarder  *MessageForwarder
	subscriber ncl.Subscriber
	msgChan    chan *ncl.Message
}

func (suite *MessageForwarderTestSuite) SetupSuite() {
	// Start a NATS server
	suite.natsServer, suite.natsConn = testutils.StartNats(suite.T())

	// Create NCL publisher
	serializer := ncl.NewEnvelopedRawMessageSerDe()
	registry := ncl.NewMessageSerDeRegistry()
	suite.Require().NoError(registry.Register(BidResultMessageType, BidResult{}))
	suite.Require().NoError(registry.Register(RunResultMessageType, RunResult{}))
	suite.Require().NoError(registry.Register(ComputeErrorMessageType, ComputeError{}))

	var err error
	suite.publisher, err = ncl.NewPublisher(
		suite.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherDestinationPrefix("test"),
		ncl.WithPublisherMessageSerializer(serializer),
		ncl.WithPublisherMessageSerDeRegistry(registry),
	)
	suite.Require().NoError(err)

	// Create MessageForwarder
	suite.forwarder = NewForwarder(suite.publisher)

	// Create NCL subscriber
	var msgHandler ncl.MessageHandlerFunc
	msgHandler = func(_ context.Context, msg *ncl.Message) error {
		suite.msgChan <- msg
		return nil
	}

	suite.msgChan = make(chan *ncl.Message, 10)
	suite.subscriber, err = ncl.NewSubscriber(
		suite.natsConn,
		ncl.WithSubscriberMessageDeserializer(serializer),
		ncl.WithSubscriberMessageSerDeRegistry(registry),
		ncl.WithSubscriberMessageHandlers(msgHandler),
	)
	suite.Require().NoError(err)
}

func (suite *MessageForwarderTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_BidAccepted() {
	execution := &models.Execution{
		ID:    "exec-1",
		JobID: "job-1",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateAskForBidAccepted,
		},
	}

	event := watcher.Event{
		Object: store.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", BidResultMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(BidResult{})
		suite.Require().True(ok)

		bidResult := result.(BidResult)
		suite.Equal(true, bidResult.Accepted)
		suite.Equal("exec-1", bidResult.ExecutionID)
		suite.Equal("job-1", bidResult.JobID)
		suite.Equal("test-job", bidResult.JobType)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_ExecutionCompleted() {
	execution := &models.Execution{
		ID:    "exec-2",
		JobID: "job-2",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateCompleted,
		},
		PublishedResult: &models.SpecConfig{Type: "myResult"},
		RunOutput:       &models.RunCommandResult{ExitCode: 0},
	}

	event := watcher.Event{
		Object: store.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", RunResultMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(RunResult{})
		suite.Require().True(ok)

		runResult := result.(RunResult)
		suite.Equal("exec-2", runResult.ExecutionID)
		suite.Equal("job-2", runResult.JobID)
		suite.Equal("test-job", runResult.JobType)
		suite.Equal("myResult", runResult.PublishResult.Type)
		suite.Equal(0, runResult.RunCommandResult.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_ExecutionFailed() {
	execution := &models.Execution{
		ID:    "exec-3",
		JobID: "job-3",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateFailed,
		},
	}

	event := watcher.Event{
		Object: store.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", ComputeErrorMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(ComputeError{})
		suite.Require().True(ok)

		computeError := result.(ComputeError)
		suite.Equal("exec-3", computeError.ExecutionID)
		suite.Equal("job-3", computeError.JobID)
		suite.Equal("test-job", computeError.JobType)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_UnhandledState() {
	execution := &models.Execution{
		ID:    "exec-4",
		JobID: "job-4",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateNew,
		},
	}

	event := watcher.Event{
		Object: store.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	// Ensure no message was published
	err = suite.subscriber.Subscribe("test.*")
	suite.Require().NoError(err)

	select {
	case <-suite.msgChan:
		suite.Fail("Received unexpected message")
	case <-time.After(100 * time.Millisecond):
		// This is the expected behavior
	}
}

func TestMessageForwarderTestSuite(t *testing.T) {
	suite.Run(t, new(MessageForwarderTestSuite))
}
