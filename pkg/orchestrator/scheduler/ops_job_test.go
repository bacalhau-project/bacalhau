//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type OpsJobSchedulerTestSuite struct {
	BaseTestSuite
	scheduler *OpsJobScheduler
}

func (s *OpsJobSchedulerTestSuite) SetupTest() {
	s.BaseTestSuite.SetupTest()
	s.scheduler = NewOpsJobScheduler(OpsJobSchedulerParams{
		JobStore:     s.jobStore,
		Planner:      s.planner,
		NodeSelector: s.nodeSelector,
	})

	// we only want to freeze time to have more deterministic tests.
	// It doesn't matter what time it is as we are using relative time to this value
	s.clock.Set(time.Now())
}

func TestOpsJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(OpsJobSchedulerTestSuite))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldCreateNewExecutions() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkLostExecutionsOnUnhealthyNodes() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the first node
	s.mockAllNodes("node1")

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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateFailed),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the first node
	s.mockAllNodes("node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_TotalTimeout() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithTotalTimeout(60*time.Minute),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
	)

	// Set the CreateTime to exceed timeout
	scenario.job.CreateTime = s.clock.Now().Add(-90 * time.Minute).UnixNano()
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
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

func (s *OpsJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithJobState(models.JobStateTypeStopped),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateCompleted),
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
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcessFail_NoMatchingNodes() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
	)
	s.mockJobStore(scenario)
	s.mockMatchingNodes(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldStopExpiredExecutions() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecutionTimeout(60*time.Minute),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateCompleted),
	)
	// Set the start time of the executions to exceed the timeout
	for i := range scenario.executions {
		scenario.executions[i].ModifyTime = s.clock.Now().Add(-90 * time.Minute).UnixNano()
	}
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
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
