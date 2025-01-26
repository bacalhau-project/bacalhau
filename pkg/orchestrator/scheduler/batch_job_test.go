//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
)

type BatchJobSchedulerTestSuite struct {
	BatchServiceJobSchedulerTestSuite
}

func (s *BatchJobSchedulerTestSuite) SetupTest() {
	s.BatchServiceJobSchedulerTestSuite.SetupTest()
	s.jobType = models.JobTypeBatch
}

func TestBatchJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchJobSchedulerTestSuite))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	scenario := NewScenario(
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)
	s.mockJobStore(scenario)
	s.mockAllNodes("node0", "node1")

	// empty plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_TooManyExecutions() {
	scenario := NewScenario(
		WithCount(2),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
		WithDesiredState(models.ExecutionDesiredStatePending),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 0), // Same partition as first one
		WithDesiredState(models.ExecutionDesiredStateRunning),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 1), // Different partition
		WithDesiredState(models.ExecutionDesiredStateStopped),
	)

	scenario.executions[1].Revision = scenario.executions[0].Revision + 1
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1")

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

func (s *BatchJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	scenario := NewScenario(
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the node in BidAccepted state
	s.mockAllNodes("node0")
	s.mockMatchingNodes(scenario, "node0")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", PartitionIndex: 1},
		},
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[1].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	scenario := NewScenario(
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateCompleted, 0),
		WithPartitionedExecution("node1", models.ExecutionStateCompleted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)
	s.mockJobStore(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoMoreNodes() {
	scenario := NewScenario(
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)
	s.mockJobStore(scenario)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.mockAllNodes()
	s.mockMatchingNodes(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[0].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
			{
				ExecutionID:  scenario.executions[1].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoRetry() {
	scenario := NewScenario(
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)
	s.mockJobStore(scenario)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
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

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_TotalTimeout() {
	scenario := NewScenario(
		WithCount(3),
		WithTotalTimeout(60*time.Minute),
		WithPartitionedExecution("node0", models.ExecutionStateBidAccepted, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
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

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldStopExpiredExecutions() {
	scenario := NewScenario(
		WithCount(3),
		WithExecutionTimeout(60*time.Minute),
		WithPartitionedExecution("node0", models.ExecutionStateBidAccepted, 0),
		WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
		WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
	)

	// Set the start time of the executions to exceed the timeout
	for i := range scenario.executions {
		scenario.executions[i].ModifyTime = s.clock.Now().Add(-90 * time.Minute).UnixNano()
	}
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1")
	s.mockMatchingNodes(scenario, "node0", "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", PartitionIndex: 0},
			{NodeID: "node1", PartitionIndex: 1},
		},
		UpdatedExecutions: []ExecutionStateUpdate{
			{
				ExecutionID:  scenario.executions[0].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
			{
				ExecutionID:  scenario.executions[1].ID,
				DesiredState: models.ExecutionDesiredStateStopped,
				ComputeState: models.ExecutionStateFailed,
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
