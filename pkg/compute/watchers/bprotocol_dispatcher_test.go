//go:build unit || !integration

package watchers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type BProtocolDispatcherTestSuite struct {
	suite.Suite
	forwarder     *BProtocolDispatcher
	mock          CallbackMock
	bidResult     legacy.BidResult
	runResult     legacy.RunResult
	computeError  legacy.ComputeError
	bidCalled     bool
	runCalled     bool
	failureCalled bool
}

func (suite *BProtocolDispatcherTestSuite) SetupTest() {
	suite.bidCalled = false
	suite.runCalled = false
	suite.failureCalled = false

	suite.mock = CallbackMock{
		OnBidCompleteHandler: func(_ context.Context, result legacy.BidResult) {
			suite.bidCalled = true
			suite.bidResult = result
		},
		OnRunCompleteHandler: func(_ context.Context, result legacy.RunResult) {
			suite.runCalled = true
			suite.runResult = result
		},
		OnComputeFailureHandler: func(_ context.Context, err legacy.ComputeError) {
			suite.failureCalled = true
			suite.computeError = err
		},
	}
	suite.forwarder = NewBProtocolDispatcher(suite.mock)
}

func (suite *BProtocolDispatcherTestSuite) createEvent(execution *models.Execution, includeEvent bool) watcher.Event {
	var events []*models.Event
	if includeEvent {
		events = append(events, &models.Event{})
	}

	return watcher.Event{
		Object: models.ExecutionUpsert{
			Current: execution,
			Events:  events,
		},
	}
}

func (suite *BProtocolDispatcherTestSuite) verifyRoutingMetadata(execution *models.Execution, metadata legacy.RoutingMetadata) {
	suite.Equal(execution.NodeID, metadata.SourcePeerID, "incorrect source peer ID")
	suite.Equal(execution.Job.OrchestratorID(), metadata.TargetPeerID, "incorrect target peer ID")
}

func (suite *BProtocolDispatcherTestSuite) verifyExecutionMetadata(execution *models.Execution, metadata legacy.ExecutionMetadata) {
	suite.Equal(execution.ID, metadata.ExecutionID, "incorrect execution ID")
	suite.Equal(execution.Job.ID, metadata.JobID, "incorrect job ID")
}

func (suite *BProtocolDispatcherTestSuite) TestHandleEvent_BidAccepted() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorID] = "requester-1"
	execution.ComputeState.StateType = models.ExecutionStateAskForBidAccepted
	event := suite.createEvent(execution, true)

	suite.Require().NoError(suite.forwarder.HandleEvent(context.Background(), event))

	suite.True(suite.bidCalled, "bid callback should be called")
	suite.False(suite.runCalled, "run callback should not be called")
	suite.False(suite.failureCalled, "failure callback should not be called")

	suite.True(suite.bidResult.Accepted)
	suite.verifyRoutingMetadata(execution, suite.bidResult.RoutingMetadata)
	suite.verifyExecutionMetadata(execution, suite.bidResult.ExecutionMetadata)
}

func (suite *BProtocolDispatcherTestSuite) TestHandleEvent_ExecutionCompleted() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorID] = "requester-1"
	execution.ComputeState.StateType = models.ExecutionStateCompleted
	execution.PublishedResult = &models.SpecConfig{Type: "myResult"}
	execution.RunOutput = &models.RunCommandResult{ExitCode: 0}
	event := suite.createEvent(execution, false)

	suite.Require().NoError(suite.forwarder.HandleEvent(context.Background(), event))

	suite.False(suite.bidCalled, "bid callback should not be called")
	suite.True(suite.runCalled, "run callback should be called")
	suite.False(suite.failureCalled, "failure callback should not be called")

	suite.Equal("myResult", suite.runResult.PublishResult.Type)
	suite.Equal(0, suite.runResult.RunCommandResult.ExitCode)
	suite.verifyRoutingMetadata(execution, suite.runResult.RoutingMetadata)
	suite.verifyExecutionMetadata(execution, suite.runResult.ExecutionMetadata)
}

func (suite *BProtocolDispatcherTestSuite) TestHandleEvent_ExecutionFailed() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorID] = "requester-1"
	execution.ComputeState.StateType = models.ExecutionStateFailed
	event := suite.createEvent(execution, true)

	suite.Require().NoError(suite.forwarder.HandleEvent(context.Background(), event))

	suite.False(suite.bidCalled, "bid callback should not be called")
	suite.False(suite.runCalled, "run callback should not be called")
	suite.True(suite.failureCalled, "failure callback should be called")

	suite.verifyRoutingMetadata(execution, suite.computeError.RoutingMetadata)
	suite.verifyExecutionMetadata(execution, suite.computeError.ExecutionMetadata)
}

func (suite *BProtocolDispatcherTestSuite) TestHandleEvent_UnhandledState() {
	execution := mock.Execution()
	execution.Job.Meta[models.MetaOrchestratorID] = "requester-1"
	execution.ComputeState.StateType = models.ExecutionStateNew
	event := suite.createEvent(execution, false)

	suite.Require().NoError(suite.forwarder.HandleEvent(context.Background(), event))

	suite.False(suite.bidCalled, "bid callback should not be called")
	suite.False(suite.runCalled, "run callback should not be called")
	suite.False(suite.failureCalled, "failure callback should not be called")
}

func TestBProtocolDispatcherTestSuite(t *testing.T) {
	suite.Run(t, new(BProtocolDispatcherTestSuite))
}
