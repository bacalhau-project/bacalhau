//go:build unit || !integration

package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
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
	suite.Require().NoError(registry.Register(compute.AskForBidMessageType, compute.AskForBidRequest{}))
	suite.Require().NoError(registry.Register(compute.BidAcceptedMessageType, compute.BidAcceptedRequest{}))
	suite.Require().NoError(registry.Register(compute.BidRejectedMessageType, compute.BidRejectedRequest{}))
	suite.Require().NoError(registry.Register(compute.CancelExecutionMessageType, compute.CancelExecutionRequest{}))

	var err error
	suite.publisher, err = ncl.NewPublisher(
		suite.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherMessageSerializer(serializer),
		ncl.WithPublisherMessageSerDeRegistry(registry),
	)
	suite.Require().NoError(err)

	// Create MessageForwarder
	subjectFn := func(nodeID string) string {
		return fmt.Sprintf("test.%s", nodeID)
	}
	suite.forwarder = NewExecutionForwarder(suite.publisher, subjectFn)

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

func (suite *MessageForwarderTestSuite) TestHandleEvent_AskForBid() {
	execution := &models.Execution{
		ID:     "exec-1",
		JobID:  "job-1",
		NodeID: "node-1",
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStatePending,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateNew,
		},
	}

	event := watcher.Event{
		Object: jobstore.ExecutionUpsert{
			Current: execution,
			Events:  []models.Event{},
		},
	}

	err := suite.subscriber.Subscribe("test.node-1.*")
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(compute.AskForBidRequest{})
		suite.Require().True(ok)

		askForBid := result.(compute.AskForBidRequest)
		suite.Equal("exec-1", askForBid.Execution.ID)
		suite.Equal("job-1", askForBid.Execution.JobID)
		suite.Equal("node-1", askForBid.Execution.NodeID)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_BidAccepted() {
	execution := &models.Execution{
		ID:     "exec-2",
		JobID:  "job-2",
		NodeID: "node-2",
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateRunning,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateAskForBidAccepted,
		},
	}

	event := watcher.Event{
		Object: jobstore.ExecutionUpsert{
			Current: execution,
			Events:  []models.Event{},
		},
	}

	err := suite.subscriber.Subscribe("test.node-2.*")
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(compute.BidAcceptedRequest{})
		suite.Require().True(ok)

		bidAccepted := result.(compute.BidAcceptedRequest)
		suite.Equal("exec-2", bidAccepted.ExecutionID)
		suite.True(bidAccepted.Accepted)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_BidRejected() {
	execution := &models.Execution{
		ID:     "exec-3",
		JobID:  "job-3",
		NodeID: "node-3",
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateStopped,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateAskForBidAccepted,
		},
	}

	event := watcher.Event{
		Object: jobstore.ExecutionUpsert{
			Current: execution,
			Events:  []models.Event{},
		},
	}

	err := suite.subscriber.Subscribe("test.node-3.*")
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(compute.BidRejectedRequest{})
		suite.Require().True(ok)

		bidRejected := result.(compute.BidRejectedRequest)
		suite.Equal("exec-3", bidRejected.ExecutionID)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_CancelExecution() {
	execution := &models.Execution{
		ID:     "exec-4",
		JobID:  "job-4",
		NodeID: "node-4",
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateStopped,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateRunning,
		},
	}

	event := watcher.Event{
		Object: jobstore.ExecutionUpsert{
			Current: execution,
			Events:  []models.Event{},
		},
	}

	err := suite.subscriber.Subscribe("test.node-4.*")
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
	suite.Require().NoError(err)

	select {
	case msg := <-suite.msgChan:
		result, ok := msg.GetPayload(compute.CancelExecutionRequest{})
		suite.Require().True(ok)

		cancelExecution := result.(compute.CancelExecutionRequest)
		suite.Equal("exec-4", cancelExecution.ExecutionID)
	case <-time.After(time.Second):
		suite.Fail("Timed out waiting for message")
	}
}

func (suite *MessageForwarderTestSuite) TestHandleEvent_UnhandledState() {
	execution := &models.Execution{
		ID:     "exec-5",
		JobID:  "job-5",
		NodeID: "node-5",
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateRunning,
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateRunning,
		},
	}

	event := watcher.Event{
		Object: jobstore.ExecutionUpsert{
			Current: execution,
			Events:  []models.Event{},
		},
	}

	err := suite.subscriber.Subscribe("*.>")
	suite.Require().NoError(err)

	err = suite.forwarder.HandleEvent(context.Background(), event)
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
