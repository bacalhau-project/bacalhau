//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
)

type ServiceJobSchedulerTestSuite struct {
	BatchServiceJobSchedulerTestSuite
}

func (s *ServiceJobSchedulerTestSuite) SetupTest() {
	s.BatchServiceJobSchedulerTestSuite.SetupTest()
}

func TestServiceSchedulerTestSuite(t *testing.T) {
	s := new(ServiceJobSchedulerTestSuite)
	s.jobType = models.JobTypeService
	suite.Run(t, s)
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)
	s.mockAllNodes("node0", "node1", "node2")

	// empty plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_TooManyExecutions() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(2),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
	)
	scenario.executions[1].Revision = scenario.executions[0].Revision + 1
	scenario.executions[2].Revision = scenario.executions[0].Revision + 1
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1", "node2")
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        scenario.evaluation,
		StoppedExecutions: []string{scenario.executions[0].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// mock node discoverer to exclude the node in BidAccepted state
	s.mockAllNodes("node0", "node3")
	s.mockMatchingNodes(scenario, "node0", "node3")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         scenario.evaluation,
		NewExecutionsNodes: []string{"node0", "node3"},
		StoppedExecutions:  []string{scenario.executions[1].ID, scenario.executions[2].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// It is a bug if a long running execution is completed. The scheduler treat those as failed executions,
// try to reschedule, or fail the job if can no longer reschedule
func (s *ServiceJobSchedulerTestSuite) TestProcess_TreatCompletedExecutionsAsFailed() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(3),
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateCompleted),
		WithExecution("node2", models.ExecutionStateAskForBid),
	)
	s.mockJobStore(scenario)
	s.mockAllNodes("node0", "node1", "node2")
	s.mockMatchingNodes(scenario, "node0", "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutionsNodes: []string{
			"node0",
			"node1",
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoMoreNodes() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
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
			scenario.executions[2].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoRetry() {
	scenario := NewScenario(
		WithJobType(models.JobTypeService),
		WithCount(3),
		WithExecution("node0", models.ExecutionStateAskForBid),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	s.mockAllNodes("node1", "node2")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			scenario.executions[0].ID,
			scenario.executions[1].ID,
			scenario.executions[2].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
