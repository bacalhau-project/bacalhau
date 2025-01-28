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
	s.mockMatchingNodes(scenario) // no more new nodes

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
	s.mockMatchingNodes(scenario) // no more new nodes

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
	s.mockMatchingNodes(scenario) // no more new nodes

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
	s.mockMatchingNodes(scenario) // no more new nodes

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
	s.mockMatchingNodes(scenario) // no more new nodes

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

func (s *OpsJobSchedulerTestSuite) TestProcess_RateLimit_InitialScheduling() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
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

func (s *OpsJobSchedulerTestSuite) TestProcess_RateLimit_WithExistingExecutions() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
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

func (s *OpsJobSchedulerTestSuite) TestProcess_RateLimit_WithUnhealthyNodes() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
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

func (s *OpsJobSchedulerTestSuite) TestProcess_NoRateLimit() {
	// Use NoopRateLimiter
	s.scheduler.rateLimiter = NewNoopRateLimiter()

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldNotRescheduleFailed() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithJobState(models.JobStateTypeRunning),
		// Have one completed and one failed execution
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateFailed),
	)
	s.mockJobStore(scenario)

	// Both nodes are still healthy
	s.mockMatchingNodes(scenario, "node0", "node1", "node2", "node3")

	// Should only create executions for new nodes, not retry failed ones
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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldNotCompleteWithPendingEvals() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecution("node0", models.ExecutionStateCompleted),
		WithExecution("node1", models.ExecutionStateCompleted),
	)
	s.mockJobStore(scenario)

	// Mock more nodes than rate limit allows
	s.mockMatchingNodes(scenario, "node2", "node3", "node4")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		// Job should stay running due to pending evals
		JobState: models.JobStateTypeRunning,
		NewExecutions: []*models.Execution{
			{NodeID: "node2", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node3", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
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

func (s *OpsJobSchedulerTestSuite) TestProcess_NoMatchingNodesWithExistingExecs() {
	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithJobState(models.JobStateTypeRunning),
		WithExecution("node0", models.ExecutionStateBidAccepted),
	)
	s.mockJobStore(scenario)

	// Mock no matching nodes but existing node is healthy
	s.mockMatchingNodes(scenario) // empty list
	s.mockAllNodes("node0")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		// No new executions since no matching nodes
		NewExecutions: []*models.Execution{},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_MixedStatesWithPendingEvals() {
	s.scheduler.rateLimiter = NewBatchRateLimiter(BatchRateLimiterParams{
		MaxExecutionsPerEval:  2,
		ExecutionLimitBackoff: 5 * time.Second,
		Clock:                 s.clock,
	})

	scenario := NewScenario(
		WithJobType(models.JobTypeOps),
		WithExecution("node0", models.ExecutionStateBidAccepted),
		WithExecution("node1", models.ExecutionStateCompleted),
		WithExecution("node2", models.ExecutionStateFailed),
	)
	s.mockJobStore(scenario)

	// Mock more new nodes than rate limit allows
	s.mockMatchingNodes(scenario, "node3", "node4", "node5", "node6")
	s.mockAllNodes("node0", "node3", "node4", "node5", "node6")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeRunning,
		NewExecutions: []*models.Execution{
			{NodeID: "node3", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
			{NodeID: "node4", DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)},
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
