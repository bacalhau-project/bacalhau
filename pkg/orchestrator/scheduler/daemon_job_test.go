//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type DaemonJobSchedulerTestSuite struct {
	BaseTestSuite
	scheduler *DaemonJobScheduler
}

func (s *DaemonJobSchedulerTestSuite) SetupTest() {
	s.BaseTestSuite.SetupTest()
	s.scheduler = NewDaemonJobScheduler(DaemonJobSchedulerParams{
		JobStore:     s.jobStore,
		Planner:      s.planner,
		NodeSelector: s.nodeSelector,
	})
}

func TestDaemonJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(DaemonJobSchedulerTestSuite))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldCreateNewExecutions() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
	)
	s.mockJobStore(scenario)
	s.mockMatchingNodes(scenario, "node0", "node1", "node2")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeRunning,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node1", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node2", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// It is a bug if a long running execution is completed. The scheduler should just ignore it
// and NOT mark the job as completed
func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldNOTMarkJobAsCompleted() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)
	s.mockMatchingNodes(scenario)

	// Noop plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldMarkLostExecutionsOnUnhealthyNodes() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the first node
	s.mockAllNodes("node1")
	s.mockMatchingNodes(scenario, "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[0].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// Even when an execution has failed, we don't mark the job as failed and continue waiting
// for more nodes that match the job selection to join.
// This requires a revisit in the future if all or a high percentage of nodes keep failing
func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldNOTMarkJobAsFailed() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateFailed),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the first node
	s.mockAllNodes("node1")
	s.mockMatchingNodes(scenario, "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[0].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithJobState(models.JobStateTypeStopped),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateAskForBidAccepted),
	)
	s.mockJobStore(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[0].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateCancelled,
			},
			{
				ExecutionID:  scenario.executions[1].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateCancelled,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcessFail_NoMatchingNodes() {
	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
	)
	s.mockJobStore(scenario)
	s.mockMatchingNodes(scenario)

	// Noop plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
