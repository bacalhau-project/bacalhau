//go:build unit || !integration

package planner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ComputeForwarderSuite struct {
	suite.Suite
	ctx              context.Context
	ctrl             *gomock.Controller
	computeService   *compute.MockEndpoint
	jobStore         *jobstore.MockStore
	nodeID           string
	plannerErr       error
	computeForwarder *ComputeForwarder
}

func TestComputeForwarderSuite(t *testing.T) {
	suite.Run(t, new(ComputeForwarderSuite))
}

func (suite *ComputeForwarderSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.ctrl = gomock.NewController(suite.T())
	suite.computeService = compute.NewMockEndpoint(suite.ctrl)
	suite.jobStore = jobstore.NewMockStore(suite.ctrl)
	suite.nodeID = "test-node"
	suite.plannerErr = errors.New("planner error")

	params := ComputeForwarderParams{
		ID:             suite.nodeID,
		ComputeService: suite.computeService,
		JobStore:       suite.jobStore,
	}

	suite.computeForwarder = NewComputeForwarder(params)
}

func (suite *ComputeForwarderSuite) TestProcess_WithNewExecutions_ShouldNotifyAskForBid() {
	plan := mock.Plan()
	execution1, execution2 := mockCreateExecutions(plan)

	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, execution1)).Times(1)
	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, execution2)).Times(1)
	suite.assertStateUpdated(execution1, model.ExecutionStateAskForBid, model.ExecutionStateNew)
	suite.assertStateUpdated(execution2, model.ExecutionStateAskForBid, model.ExecutionStateNew)
	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) TestProcess_WithUpdatedExecutions_ShouldNotifyAppropriateStates() {
	plan := mock.Plan()
	toAskForBid := suite.mockUpdateExecution(plan, "toAskForBid", model.ExecutionDesiredStatePending, model.ExecutionStateNew)
	bidAccepted := suite.mockUpdateExecution(plan, "bidAccepted", model.ExecutionDesiredStateRunning, model.ExecutionStateAskForBidAccepted)
	bidRejected := suite.mockUpdateExecution(plan, "bidRejected", model.ExecutionDesiredStateStopped, model.ExecutionStateAskForBidAccepted)
	toCancel1 := suite.mockUpdateExecution(plan, "toCancel1", model.ExecutionDesiredStateStopped, model.ExecutionStateNew)
	toCancel2 := suite.mockUpdateExecution(plan, "toCancel2", model.ExecutionDesiredStateStopped, model.ExecutionStateAskForBid)
	toCancel3 := suite.mockUpdateExecution(plan, "toCancel3", model.ExecutionDesiredStateStopped, model.ExecutionStateBidAccepted)

	// noop
	suite.mockUpdateExecution(plan, "noop1", model.ExecutionDesiredStateStopped, model.ExecutionStateFailed)
	suite.mockUpdateExecution(plan, "noop2", model.ExecutionDesiredStateStopped, model.ExecutionStateCompleted)

	// NotifyAskForBid
	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toAskForBid.Execution)).Times(1)
	suite.assertStateUpdated(toAskForBid.Execution, model.ExecutionStateAskForBid, model.ExecutionStateNew)
	// NotifyBidAccepted
	suite.computeService.EXPECT().BidAccepted(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, bidAccepted.Execution)).Times(1)
	suite.assertStateUpdated(bidAccepted.Execution, model.ExecutionStateBidAccepted, model.ExecutionStateAskForBidAccepted)

	// NotifyBidRejected
	suite.computeService.EXPECT().BidRejected(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, bidRejected.Execution)).Times(1)
	suite.assertStateUpdated(bidRejected.Execution, model.ExecutionStateBidRejected, model.ExecutionStateAskForBidAccepted)

	// NotifyCancel
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toCancel1.Execution)).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toCancel2.Execution)).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toCancel3.Execution)).Times(1)
	suite.assertStateUpdated(toCancel1.Execution, model.ExecutionStateCancelled, model.ExecutionStateUndefined)
	suite.assertStateUpdated(toCancel2.Execution, model.ExecutionStateCancelled, model.ExecutionStateUndefined)
	suite.assertStateUpdated(toCancel3.Execution, model.ExecutionStateCancelled, model.ExecutionStateUndefined)

	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) TestProcess_OnNotifyFailure_NoStateUpdate() {
	plan := mock.Plan()
	toAskForBid := suite.mockUpdateExecution(plan, "toAskForBid", model.ExecutionDesiredStatePending, model.ExecutionStateNew)
	bidAccepted := suite.mockUpdateExecution(plan, "bidAccepted", model.ExecutionDesiredStateRunning, model.ExecutionStateAskForBidAccepted)
	bidRejected := suite.mockUpdateExecution(plan, "bidRejected", model.ExecutionDesiredStateStopped, model.ExecutionStateAskForBidAccepted)
	toCancel1 := suite.mockUpdateExecution(plan, "toCancel1", model.ExecutionDesiredStateStopped, model.ExecutionStateNew)

	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toAskForBid.Execution)).Return(compute.AskForBidResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().BidAccepted(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, bidAccepted.Execution)).Return(compute.BidAcceptedResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().BidRejected(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, bidRejected.Execution)).Return(compute.BidRejectedResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, toCancel1.Execution)).Return(compute.CancelExecutionResponse{}, suite.plannerErr).Times(1)
	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) mockUpdateExecution(plan *models.Plan, id string, desiredState model.ExecutionDesiredState, currentState model.ExecutionStateType) *models.PlanExecutionDesiredUpdate {
	execution := mock.ExecutionState(plan.Job.ID())
	execution.ComputeReference = id
	execution.State = currentState
	update := &models.PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: desiredState,
		Comment:      "update",
	}
	plan.UpdatedExecutions[execution.ID()] = update
	return update

}
func (suite *ComputeForwarderSuite) assertStateUpdated(execution *model.ExecutionState, newState model.ExecutionStateType, expectedState model.ExecutionStateType) {
	matcher := NewUpdateExecutionMatcher(suite.T(), execution, UpdateExecutionMatcherParams{
		NewState:      newState,
		ExpectedState: expectedState,
	})
	suite.jobStore.EXPECT().UpdateExecution(suite.ctx, matcher).Times(1)
}

func (suite *ComputeForwarderSuite) waitUntilSatisfied() bool {
	return suite.Eventually(func() bool {
		return suite.ctrl.Satisfied()
	}, 500*time.Millisecond, 10*time.Millisecond)
}
