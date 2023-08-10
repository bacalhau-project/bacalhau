//go:build unit || !integration

package planner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
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
	suite.assertStateUpdated(execution1, models.ExecutionStateBidAccepted, models.ExecutionStateNew)
	suite.assertStateUpdated(execution2, models.ExecutionStateBidAccepted, models.ExecutionStateNew)
	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) TestProcess_WithNewExecutions_ShouldNotifyAskForBid_Pending() {
	plan := mock.Plan()
	execution1, execution2 := mockCreateExecutions(plan)
	execution1.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)
	execution2.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)

	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, execution1)).Times(1)
	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcher(suite.T(), suite.nodeID, execution2)).Times(1)
	suite.assertStateUpdated(execution1, models.ExecutionStateAskForBid, models.ExecutionStateNew)
	suite.assertStateUpdated(execution2, models.ExecutionStateAskForBid, models.ExecutionStateNew)
	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) TestProcess_WithUpdatedExecutions_ShouldNotifyAppropriateStates() {
	plan := mock.Plan()
	toAskForBid := suite.mockUpdateExecution(plan, "toAskForBid", models.ExecutionDesiredStateRunning, models.ExecutionStateNew)
	toAskForBidPending := suite.mockUpdateExecution(plan, "toAskForBidPending", models.ExecutionDesiredStatePending, models.ExecutionStateNew)
	bidAccepted := suite.mockUpdateExecution(plan, "bidAccepted", models.ExecutionDesiredStateRunning, models.ExecutionStateAskForBidAccepted)
	bidRejected := suite.mockUpdateExecution(plan, "bidRejected", models.ExecutionDesiredStateStopped, models.ExecutionStateAskForBidAccepted)
	toCancel1 := suite.mockUpdateExecution(plan, "toCancel1", models.ExecutionDesiredStateStopped, models.ExecutionStateNew)
	toCancel2 := suite.mockUpdateExecution(plan, "toCancel2", models.ExecutionDesiredStateStopped, models.ExecutionStateAskForBid)
	toCancel3 := suite.mockUpdateExecution(plan, "toCancel3", models.ExecutionDesiredStateStopped, models.ExecutionStateBidAccepted)

	// noop
	suite.mockUpdateExecution(plan, "noop1", models.ExecutionDesiredStateStopped, models.ExecutionStateFailed)
	suite.mockUpdateExecution(plan, "noop2", models.ExecutionDesiredStateStopped, models.ExecutionStateCompleted)

	// NotifyAskForBid
	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toAskForBid)).Times(1)
	suite.assertStateUpdated(toAskForBid.Execution, models.ExecutionStateBidAccepted, models.ExecutionStateNew)
	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toAskForBidPending)).Times(1)
	suite.assertStateUpdated(toAskForBidPending.Execution, models.ExecutionStateAskForBid, models.ExecutionStateNew)
	// NotifyBidAccepted
	suite.computeService.EXPECT().BidAccepted(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, bidAccepted)).Times(1)
	suite.assertStateUpdated(bidAccepted.Execution, models.ExecutionStateBidAccepted, models.ExecutionStateAskForBidAccepted)

	// NotifyBidRejected
	suite.computeService.EXPECT().BidRejected(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, bidRejected)).Times(1)
	suite.assertStateUpdated(bidRejected.Execution, models.ExecutionStateBidRejected, models.ExecutionStateAskForBidAccepted)

	// NotifyCancel
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toCancel1)).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toCancel2)).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toCancel3)).Times(1)
	suite.assertStateUpdated(toCancel1.Execution, models.ExecutionStateCancelled, models.ExecutionStateUndefined)
	suite.assertStateUpdated(toCancel2.Execution, models.ExecutionStateCancelled, models.ExecutionStateUndefined)
	suite.assertStateUpdated(toCancel3.Execution, models.ExecutionStateCancelled, models.ExecutionStateUndefined)

	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) TestProcess_OnNotifyFailure_NoStateUpdate() {
	plan := mock.Plan()
	toAskForBid := suite.mockUpdateExecution(plan, "toAskForBid", models.ExecutionDesiredStatePending, models.ExecutionStateNew)
	bidAccepted := suite.mockUpdateExecution(plan, "bidAccepted", models.ExecutionDesiredStateRunning, models.ExecutionStateAskForBidAccepted)
	bidRejected := suite.mockUpdateExecution(plan, "bidRejected", models.ExecutionDesiredStateStopped, models.ExecutionStateAskForBidAccepted)
	toCancel1 := suite.mockUpdateExecution(plan, "toCancel1", models.ExecutionDesiredStateStopped, models.ExecutionStateNew)

	suite.computeService.EXPECT().AskForBid(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toAskForBid)).Return(compute.AskForBidResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().BidAccepted(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, bidAccepted)).Return(compute.BidAcceptedResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().BidRejected(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, bidRejected)).Return(compute.BidRejectedResponse{}, suite.plannerErr).Times(1)
	suite.computeService.EXPECT().CancelExecution(suite.ctx, NewComputeRequestMatcherFromPlanUpdate(suite.T(), suite.nodeID, toCancel1)).Return(compute.CancelExecutionResponse{}, suite.plannerErr).Times(1)
	suite.NoError(suite.computeForwarder.Process(suite.ctx, plan))

	suite.waitUntilSatisfied()
}

func (suite *ComputeForwarderSuite) mockUpdateExecution(plan *models.Plan, id string, desiredState models.ExecutionDesiredStateType, currentState models.ExecutionStateType) *models.PlanExecutionDesiredUpdate {
	execution := mock.Execution(plan.Job)
	execution.ID = id
	execution.ComputeState = models.NewExecutionState(currentState)
	update := &models.PlanExecutionDesiredUpdate{
		Execution:    execution,
		DesiredState: desiredState,
		Comment:      "update",
	}
	plan.UpdatedExecutions[execution.ID] = update
	return update

}
func (suite *ComputeForwarderSuite) assertStateUpdated(execution *models.Execution, newState models.ExecutionStateType, expectedState models.ExecutionStateType) {
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
