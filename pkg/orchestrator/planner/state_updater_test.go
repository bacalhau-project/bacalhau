//go:build unit || !integration

package planner

import (
	"context"
	"errors"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type StateUpdaterSuite struct {
	suite.Suite
	ctrl         *gomock.Controller
	ctx          context.Context
	mockStore    *jobstore.MockStore
	stateUpdater *StateUpdater
}

func (suite *StateUpdaterSuite) SetupTest() {
	suite.ctx = context.Background()
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockStore = jobstore.NewMockStore(suite.ctrl)
	suite.stateUpdater = NewStateUpdater(suite.mockStore)
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_CreateExecutions_Success() {
	plan := mock.Plan()
	execution1, execution2 := mockCreateExecutions(plan)

	suite.mockStore.EXPECT().CreateExecution(suite.ctx, *execution1).Times(1)
	suite.mockStore.EXPECT().CreateExecution(suite.ctx, *execution2).Times(1)

	suite.NoError(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_CreateExecutions_Error() {
	plan := mock.Plan()
	execution1, _ := mockCreateExecutions(plan)

	// no attempt to create execution2
	suite.mockStore.EXPECT().CreateExecution(suite.ctx, *execution1).Return(errors.New("create error")).Times(1)

	suite.Error(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_UpdateExecutions_Success() {
	plan := mock.Plan()
	update1, update2 := mockUpdateExecutions(plan)

	suite.mockStore.EXPECT().UpdateExecution(suite.ctx, NewUpdateExecutionMatcherFromPlanUpdate(suite.T(), update1)).Times(1)
	suite.mockStore.EXPECT().UpdateExecution(suite.ctx, NewUpdateExecutionMatcherFromPlanUpdate(suite.T(), update2)).Times(1)
	suite.NoError(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_UpdateJobState_Success() {
	plan := mock.Plan()
	plan.DesiredJobState = models.JobStateTypeCompleted
	plan.Comment = "update job state"

	suite.mockStore.EXPECT().UpdateJobState(suite.ctx, NewUpdateJobMatcherFromPlanUpdate(suite.T(), plan)).Times(1)
	suite.NoError(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_UpdateJobState_Error() {
	plan := mock.Plan()
	plan.DesiredJobState = models.JobStateTypeCompleted
	plan.Comment = "update job state"

	suite.mockStore.EXPECT().UpdateJobState(suite.ctx, NewUpdateJobMatcherFromPlanUpdate(suite.T(), plan)).Return(errors.New("create error")).Times(1)
	suite.Error(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_NoOp() {
	plan := mock.Plan()
	suite.NoError(suite.stateUpdater.Process(suite.ctx, plan))
}

func (suite *StateUpdaterSuite) TestStateUpdater_Process_MultiOp() {
	plan := mock.Plan()
	execution1, execution2 := mockCreateExecutions(plan)
	update1, update2 := mockUpdateExecutions(plan)
	plan.DesiredJobState = models.JobStateTypeCompleted
	plan.Comment = "update job state"

	suite.mockStore.EXPECT().CreateExecution(suite.ctx, *execution1).Times(1)
	suite.mockStore.EXPECT().CreateExecution(suite.ctx, *execution2).Times(1)
	suite.mockStore.EXPECT().UpdateExecution(suite.ctx, NewUpdateExecutionMatcherFromPlanUpdate(suite.T(), update1)).Times(1)
	suite.mockStore.EXPECT().UpdateExecution(suite.ctx, NewUpdateExecutionMatcherFromPlanUpdate(suite.T(), update2)).Times(1)
	suite.mockStore.EXPECT().UpdateJobState(suite.ctx, NewUpdateJobMatcherFromPlanUpdate(suite.T(), plan)).Times(1)

	suite.NoError(suite.stateUpdater.Process(suite.ctx, plan))
}

func TestStateUpdaterSuite(t *testing.T) {
	suite.Run(t, new(StateUpdaterSuite))
}
