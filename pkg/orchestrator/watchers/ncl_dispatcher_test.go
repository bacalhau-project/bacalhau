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
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type NCLDispatcherTestSuite struct {
	suite.Suite
	natsServer *server.Server
	natsConn   *nats.Conn
	publisher  ncl.Publisher
	jobStore   *jobstore.MockStore
	dispatcher *NCLDispatcher
	subscriber ncl.Subscriber
	msgChan    chan *ncl.Message
}

// Setup and TearDown methods remain the same...
func (s *NCLDispatcherTestSuite) SetupSuite() {
	// Start a NATS server
	s.natsServer, s.natsConn = testutils.StartNats(s.T())

	// Create NCL publisher
	serializer := ncl.NewEnvelopedRawMessageSerDe()
	registry := ncl.NewMessageSerDeRegistry()
	s.Require().NoError(registry.Register(messages.AskForBidMessageType, messages.AskForBidRequest{}))
	s.Require().NoError(registry.Register(messages.BidAcceptedMessageType, messages.BidAcceptedRequest{}))
	s.Require().NoError(registry.Register(messages.BidRejectedMessageType, messages.BidRejectedRequest{}))
	s.Require().NoError(registry.Register(messages.CancelExecutionMessageType, messages.CancelExecutionRequest{}))

	var err error
	s.publisher, err = ncl.NewPublisher(
		s.natsConn,
		ncl.WithPublisherName("test-publisher"),
		ncl.WithPublisherMessageSerializer(serializer),
		ncl.WithPublisherMessageSerDeRegistry(registry),
	)
	s.Require().NoError(err)

	// Add mock jobstore
	s.jobStore = jobstore.NewMockStore(gomock.NewController(s.T()))

	// Create NCLDispatcher with jobstore
	subjectFn := func(nodeID string) string {
		return fmt.Sprintf("test.%s", nodeID)
	}
	s.dispatcher = NewNCLDispatcher(NCLDispatcherParams{
		Publisher: s.publisher,
		SubjectFn: subjectFn,
		JobStore:  s.jobStore,
	})

	// Create NCL subscriber
	var msgHandler ncl.MessageHandlerFunc
	msgHandler = func(_ context.Context, msg *ncl.Message) error {
		s.msgChan <- msg
		return nil
	}

	s.msgChan = make(chan *ncl.Message, 10)
	s.subscriber, err = ncl.NewSubscriber(
		s.natsConn,
		ncl.WithSubscriberMessageDeserializer(serializer),
		ncl.WithSubscriberMessageSerDeRegistry(registry),
		ncl.WithSubscriberMessageHandlers(msgHandler),
	)
	s.Require().NoError(err)
}

func (s *NCLDispatcherTestSuite) TearDownSuite() {
	s.natsConn.Close()
	s.natsServer.Shutdown()
}

func (s *NCLDispatcherTestSuite) TestHandleEvent_AskForBid() {
	tests := []struct {
		name            string
		desiredState    models.ExecutionDesiredStateType
		waitForApproval bool
	}{
		{
			name:            "pending_state",
			desiredState:    models.ExecutionDesiredStatePending,
			waitForApproval: true,
		},
		{
			name:            "running_state",
			desiredState:    models.ExecutionDesiredStateRunning,
			waitForApproval: false,
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			upsert := setupNewExecution(
				tc.desiredState,
				models.ExecutionStateNew,
			)
			subject := fmt.Sprintf("test.%s.*", upsert.Current.NodeID)

			s.Require().NoError(s.setupAndHandleEvent(upsert, subject))

			s.assertMessageReceived(messages.AskForBidRequest{}, func(result interface{}) {
				askForBid := result.(messages.AskForBidRequest)
				s.Equal(upsert.Current.ID, askForBid.Execution.ID)
				s.Equal(upsert.Current.JobID, askForBid.Execution.JobID)
				s.Equal(upsert.Current.NodeID, askForBid.Execution.NodeID)
			})
		})
	}
}

func (s *NCLDispatcherTestSuite) TestHandleEvent_BidAccepted() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateAskForBidAccepted,
	)
	subject := fmt.Sprintf("test.%s.*", upsert.Current.NodeID)

	s.Require().NoError(s.setupAndHandleEvent(upsert, subject))

	s.assertMessageReceived(messages.BidAcceptedRequest{}, func(result interface{}) {
		bidAccepted := result.(messages.BidAcceptedRequest)
		s.Equal(upsert.Current.ID, bidAccepted.ExecutionID)
		s.True(bidAccepted.Accepted)
	})
}

func (s *NCLDispatcherTestSuite) TestHandleEvent_BidRejected() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStatePending,
		models.ExecutionStateAskForBidAccepted,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateAskForBidAccepted,
	)
	subject := fmt.Sprintf("test.%s.*", upsert.Current.NodeID)

	s.Require().NoError(s.setupAndHandleEvent(upsert, subject))

	s.assertMessageReceived(messages.BidRejectedRequest{}, func(result interface{}) {
		bidRejected := result.(messages.BidRejectedRequest)
		s.Equal(upsert.Current.ID, bidRejected.ExecutionID)
	})
}

func (s *NCLDispatcherTestSuite) TestHandleEvent_CancelExecution() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateStopped,
		models.ExecutionStateRunning,
	)
	subject := fmt.Sprintf("test.%s.*", upsert.Current.NodeID)

	// Expect jobstore update when cancelling
	s.jobStore.EXPECT().UpdateExecution(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, req jobstore.UpdateExecutionRequest) error {
			s.Equal(upsert.Current.ID, req.ExecutionID)
			s.Equal(models.ExecutionStateCancelled, req.NewValues.ComputeState.StateType)
			return nil
		})

	s.Require().NoError(s.setupAndHandleEvent(upsert, subject))

	s.assertMessageReceived(messages.CancelExecutionRequest{}, func(result interface{}) {
		cancelExecution := result.(messages.CancelExecutionRequest)
		s.Equal(upsert.Current.ID, cancelExecution.ExecutionID)
	})
}

func (s *NCLDispatcherTestSuite) TestHandleEvent_UnhandledState() {
	upsert := setupStateTransition(
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
		models.ExecutionDesiredStateRunning,
		models.ExecutionStateRunning,
	)

	s.Require().NoError(s.setupAndHandleEvent(upsert, "*.>"))
	s.assertNoMessage()
}

// setupAndHandleEvent sets up a test case by subscribing to the appropriate NATS subject and creating the event
func (s *NCLDispatcherTestSuite) setupAndHandleEvent(upsert models.ExecutionUpsert, subject string) error {
	if err := s.subscriber.Subscribe(subject); err != nil {
		return err
	}
	return s.dispatcher.HandleEvent(context.Background(), createExecutionEvent(upsert))
}

// assertMessageReceived verifies that the expected message type and content is received
func (s *NCLDispatcherTestSuite) assertMessageReceived(expectedType interface{}, validate func(interface{})) {
	select {
	case msg := <-s.msgChan:
		result, ok := msg.GetPayload(expectedType)
		s.Require().True(ok)
		validate(result)
	case <-time.After(time.Second):
		s.Fail("Timed out waiting for message")
	}
}

// assertNoMessage verifies that no message is received within the timeout
func (s *NCLDispatcherTestSuite) assertNoMessage() {
	select {
	case <-s.msgChan:
		s.Fail("Received unexpected message")
	case <-time.After(100 * time.Millisecond):
		// Expected behavior
	}
}

func TestNCLDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(NCLDispatcherTestSuite))
}

// setupNewExecution creates an upsert for a new execution with no previous state
func setupNewExecution(
	desiredState models.ExecutionDesiredStateType,
	computeState models.ExecutionStateType,
	events ...*models.Event,
) models.ExecutionUpsert {
	execution := mock.Execution()
	execution.ComputeState = models.NewExecutionState(computeState)
	execution.DesiredState = models.NewExecutionDesiredState(desiredState)

	return models.ExecutionUpsert{
		Previous: nil,
		Current:  execution,
		Events:   events,
	}
}

// setupStateTransition creates an upsert for an execution state transition
func setupStateTransition(
	prevDesiredState models.ExecutionDesiredStateType,
	prevComputeState models.ExecutionStateType,
	newDesiredState models.ExecutionDesiredStateType,
	newComputeState models.ExecutionStateType,
	events ...*models.Event,
) models.ExecutionUpsert {
	previous := mock.Execution()
	previous.ComputeState = models.NewExecutionState(prevComputeState)
	previous.DesiredState = models.NewExecutionDesiredState(prevDesiredState)

	current := mock.Execution()
	current.ID = previous.ID // Ensure same execution
	current.JobID = previous.JobID
	current.NodeID = previous.NodeID
	current.ComputeState = models.NewExecutionState(newComputeState)
	current.DesiredState = models.NewExecutionDesiredState(newDesiredState)

	return models.ExecutionUpsert{
		Previous: previous,
		Current:  current,
		Events:   events,
	}
}

// createExecutionEvent is a helper to create watcher.Event from an ExecutionUpsert
func createExecutionEvent(upsert models.ExecutionUpsert) watcher.Event {
	return watcher.Event{
		Object: upsert,
	}
}
