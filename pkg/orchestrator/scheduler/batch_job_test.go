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
}

func TestBatchJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchJobSchedulerTestSuite))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	scenario := NewScenario(
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
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
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
	)
	scenario.executions[1].Revision = scenario.executions[0].Revision + 1
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1")
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        scenario.evaluation,
		StoppedExecutions: []string{scenario.executions[0].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	scenario := NewScenario(
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the node in BidAccepted state
	s.mockAllNodes("node0")
	s.mockMatchingNodes(scenario, "node0")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         scenario.evaluation,
		NewExecutionsNodes: []string{"node0"},
		StoppedExecutions:  []string{scenario.executions[1].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	scenario := NewScenario(
		WithCount(3),
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateCompleted),
		WithExecution("node2", models.ExecutionStateCompleted),
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
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.mockAllNodes()
	s.mockMatchingNodes(scenario)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			scenario.executions[0].ID,
			scenario.executions[1].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoRetry() {
	scenario := NewScenario(
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
	s.mockAllNodes("node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			scenario.executions[0].ID,
			scenario.executions[1].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_TotalTimeout() {
	scenario := NewScenario(
		WithCount(3),
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
		StoppedExecutions: []string{
			scenario.executions[0].ID,
			scenario.executions[1].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldStopExpiredExecutions() {
	scenario := NewScenario(
		WithCount(3),
		WithExecutionTimeout(60*time.Minute),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateCompleted),
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
		Evaluation:         scenario.evaluation,
		NewExecutionsNodes: []string{"node0", "node1"},
		StoppedExecutions: []string{
			scenario.executions[0].ID,
			scenario.executions[1].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
