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

func (s *DaemonJobSchedulerTestSuite) TestProcess_RateLimit_InitialScheduling() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
	)
	s.mockJobStore(scenario)

	// Mock that we have 4 matching nodes
	s.mockMatchingNodes(scenario, "node0", "node1", "node2", "node3")

	// Should only create 2 executions and schedule a delayed evaluation for the rest
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeRunning,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node1", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
		},
		ExpectedNewEvaluations: []ExpectedEvaluation{
			{
				TriggeredBy: models.EvalTriggerExecutionLimit,
				WaitUntil:   s.clock.Now().Add(5 * time.Second),
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_RateLimit_WithExistingExecutions() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithJobState(models.JobStateTypeRunning),
		// Already have executions on two nodes
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// Mock that we have 4 matching nodes, 2 existing and 2 new
	s.mockAllNodes("node0", "node1", "node2", "node3")
	s.mockMatchingNodes(scenario, "node0", "node1", "node2", "node3")

	// Should only create executions for the 2 new nodes within rate limit
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node2", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node3", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_RateLimit_WithSomeUnhealthyNodes() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithJobState(models.JobStateTypeRunning),
		// Have existing executions on three nodes
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateBidAccepted),
		WithExecution("node2", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// node0 and node1 became unhealthy, but we have 3 new healthy nodes
	s.mockAllNodes("node2", "node3", "node4", "node5")
	s.mockMatchingNodes(scenario, "node2", "node3", "node4", "node5")

	// Should:
	// 1. Mark node0 and node1's executions as failed
	// 2. Create 2 new executions (limited by rate limiter)
	// 3. Schedule delayed evaluation for the remaining new node
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node3", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node4", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
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
		ExpectedNewEvaluations: []ExpectedEvaluation{
			{
				TriggeredBy: models.EvalTriggerExecutionLimit,
				WaitUntil:   s.clock.Now().Add(5 * time.Second),
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_NoRateLimit() {
	// Use NoopRateLimiter
	s.scheduler.rateLimiter = NewNoopRateLimiter()

	scenario := NewScenario(
		WithJobType(models.JobTypeDaemon),
		WithJobState(models.JobStateTypeRunning),
		// Have one existing execution
		WithExecution("node0", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// Mock that we have 4 total matching nodes
	s.mockAllNodes("node0", "node1", "node2", "node3")
	s.mockMatchingNodes(scenario, "node0", "node1", "node2", "node3")

	// Should create executions for all new nodes at once
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node1", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node2", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node3", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
