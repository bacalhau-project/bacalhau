//go:build unit || !integration

package watchers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type NCLDispatcherTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	publisher  ncl.Publisher
	forwarder  *NCLDispatcher
	subscriber ncl.Subscriber
	msgChan    chan *envelope.Message
}

func (suite *NCLDispatcherTestSuite) SetupSuite() {
	// Start a NATS server
	suite.natsServer, suite.natsConn = testutils.StartNats(suite.T())

	// Create NCL publisher
	serializer := envelope.NewSerializer()
	registry := envelope.NewRegistry()
	suite.Require().NoError(registry.Register(messages.BidResultMessageType, messages.BidResult{}))
	suite.Require().NoError(registry.Register(messages.RunResultMessageType, messages.RunResult{}))
	suite.Require().NoError(registry.Register(messages.ComputeErrorMessageType, messages.ComputeError{}))

	var err error
	suite.publisher, err = ncl.NewPublisher(
		suite.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherDestinationPrefix("test"),
		ncl.WithPublisherMessageSerializer(serializer),
		ncl.WithPublisherMessageSerDeRegistry(registry),
	)
	suite.Require().NoError(err)

	// Create NCLDispatcher
	suite.forwarder = NewNCLDispatcher(suite.publisher)

	// Create NCL subscriber
	var msgHandler ncl.MessageHandlerFunc
	msgHandler = func(_ context.Context, msg *envelope.Message) error {
		suite.msgChan <- msg
		return nil
	}

	suite.msgChan = make(chan *envelope.Message, 10)
	suite.subscriber, err = ncl.NewSubscriber(
		suite.natsConn,
		ncl.WithSubscriberMessageDeserializer(serializer),
		ncl.WithSubscriberMessageSerDeRegistry(registry),
		ncl.WithSubscriberMessageHandlers(msgHandler),
	)
	suite.Require().NoError(err)
}

func (suite *NCLDispatcherTestSuite) TearDownSuite() {
	suite.natsConn.Close()
	suite.natsServer.Shutdown()
}

func (suite *NCLDispatcherTestSuite) TestHandleEvent_BidAccepted() {
	execution := &models.Execution{
		ID:    "exec-1",
		JobID: "job-1",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateAskForBidAccepted,
		},
	}

	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", messages.BidResultMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(messages.BidResult{})
		suite.Require().True(ok)

		bidResult := result.(messages.BidResult)
		suite.Equal(true, bidResult.Accepted)
		suite.Equal("exec-1", bidResult.ExecutionID)
		suite.Equal("job-1", bidResult.JobID)
		suite.Equal("test-job", bidResult.JobType)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *NCLDispatcherTestSuite) TestHandleEvent_ExecutionCompleted() {
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
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", messages.RunResultMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(messages.RunResult{})
		suite.Require().True(ok)

		runResult := result.(messages.RunResult)
		suite.Equal("exec-2", runResult.ExecutionID)
		suite.Equal("job-2", runResult.JobID)
		suite.Equal("test-job", runResult.JobType)
		suite.Equal("myResult", runResult.PublishResult.Type)
		suite.Equal(0, runResult.RunCommandResult.ExitCode)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *NCLDispatcherTestSuite) TestHandleEvent_ExecutionFailed() {
	execution := &models.Execution{
		ID:    "exec-3",
		JobID: "job-3",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateFailed,
		},
	}

	event := watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  []*models.Event{},
		},
	}

	err := suite.subscriber.Subscribe(fmt.Sprintf("test.%s", messages.ComputeErrorMessageType))
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(messages.ComputeError{})
		suite.Require().True(ok)

		computeError := result.(messages.ComputeError)
		suite.Equal("exec-3", computeError.ExecutionID)
		suite.Equal("job-3", computeError.JobID)
		suite.Equal("test-job", computeError.JobType)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *NCLDispatcherTestSuite) TestHandleEvent_UnhandledState() {
	execution := &models.Execution{
		ID:    "exec-4",
		JobID: "job-4",
		Job:   &models.Job{Type: "test-job"},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateNew,
		},
	}

	event := watcher.Event{
		Object: models.ExecutionUpsert{
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

func TestNCLDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(NCLDispatcherTestSuite))
}
