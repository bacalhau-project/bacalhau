//go:build unit || !integration

package planner

import (
	"context"
	"errors"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type ChainedPlannerSuite struct {
	suite.Suite
	ctrl       *gomock.Controller
	planner1   *orchestrator.MockPlanner
	planner2   *orchestrator.MockPlanner
	planner3   *orchestrator.MockPlanner
	plannerErr error
}

func (suite *ChainedPlannerSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.planner1 = orchestrator.NewMockPlanner(suite.ctrl)
	suite.planner2 = orchestrator.NewMockPlanner(suite.ctrl)
	suite.planner3 = orchestrator.NewMockPlanner(suite.ctrl)
	suite.plannerErr = errors.New("planner error")
}

func (suite *ChainedPlannerSuite) TestChainedPlanner_Process() {
	chainedPlanner := NewChain(suite.planner1, suite.planner2, suite.planner3)

	// Create a sample plan
	plan := mock.Plan()

	// Set up expectations
	suite.planner1.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)
	suite.planner2.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)
	suite.planner3.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)

	err := chainedPlanner.Process(context.Background(), plan)

	// Ensure that all planners were invoked
	suite.NoError(err)
}

func (suite *ChainedPlannerSuite) TestChainedPlanner_Process_NoPlanners() {
	chainedPlanner := NewChain()

	// Create a sample plan
	plan := mock.Plan()

	err := chainedPlanner.Process(context.Background(), plan)

	// Ensure that no errors occurred
	suite.NoError(err)
}

func (suite *ChainedPlannerSuite) TestChainedPlanner_Process_PlannerError() {
	chainedPlanner := NewChain(suite.planner1, suite.planner2, suite.planner3)

	// Create a sample plan
	plan := mock.Plan()

	// Set up expectations
	suite.planner1.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)
	suite.planner2.EXPECT().Process(gomock.Any(), plan).Return(suite.plannerErr).Times(1)
	suite.planner3.EXPECT().Process(gomock.Any(), plan).Times(0)

	err := chainedPlanner.Process(context.Background(), plan)

	// Ensure that the error from the failed planner is returned
	suite.ErrorContains(err, suite.plannerErr.Error())
}

func (suite *ChainedPlannerSuite) TestChainedPlanner_Process_Order() {
	chainedPlanner := NewChain(suite.planner1, suite.planner2, suite.planner3)

	// Create a sample plan
	plan := mock.Plan()

	// Set up expectations
	gomock.InOrder(
		suite.planner1.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1),
		suite.planner2.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1),
		suite.planner3.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1),
	)

	err := chainedPlanner.Process(context.Background(), plan)

	// Ensure that all planners were invoked in order
	suite.NoError(err)
}

func (suite *ChainedPlannerSuite) TestChainedPlanner_Add() {
	chainedPlanner := NewChain()

	// Add additional planners
	chainedPlanner.Add(suite.planner1)
	chainedPlanner.Add(suite.planner2)

	// Create a sample plan
	plan := mock.Plan()

	// Set up expectations for the added planners
	suite.planner1.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)
	suite.planner2.EXPECT().Process(gomock.Any(), plan).Return(nil).Times(1)

	err := chainedPlanner.Process(context.Background(), plan)

	// Ensure that all planners were invoked
	suite.NoError(err)
}

func TestChainedPlannerSuite(t *testing.T) {
	suite.Run(t, new(ChainedPlannerSuite))
}
