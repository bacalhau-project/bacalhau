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

type BatchServiceJobSchedulerTestSuite struct {
	BaseTestSuite
	scheduler *BatchServiceJobScheduler
	jobType   string
}

// NewBatchServiceJobSchedulerTestSuite accepts jobType and returns new instance
func NewBatchServiceJobSchedulerTestSuite(jobType string) *BatchServiceJobSchedulerTestSuite {
	return &BatchServiceJobSchedulerTestSuite{
		jobType: jobType,
	}
}

func (s *BatchServiceJobSchedulerTestSuite) SetupTest() {
	s.BaseTestSuite.SetupTest()
	s.scheduler = NewBatchServiceJobScheduler(BatchServiceJobSchedulerParams{
		JobStore:      s.jobStore,
		Planner:       s.planner,
		NodeSelector:  s.nodeSelector,
		RetryStrategy: s.retryStrategy,
		QueueBackoff:  5 * time.Second,
		Clock:         s.clock,
	})
}

func TestBatchServiceJobSchedulerTestSuiteBatch(t *testing.T) {
	suite.Run(t, NewBatchServiceJobSchedulerTestSuite(models.JobTypeBatch))
}

func TestBatchServiceJobSchedulerTestSuiteService(t *testing.T) {
	suite.Run(t, NewBatchServiceJobSchedulerTestSuite(models.JobTypeService))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldCreateEnoughExecutions() {
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(3),
	)
	s.mockJobStore(scenario)

	// we need 3 executions. discover enough nodes
	s.mockMatchingNodes(scenario, "node0", "node1", "node2", "node3")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", PartitionIndex: 0},
			{NodeID: "node1", PartitionIndex: 1},
			{NodeID: "node2", PartitionIndex: 2},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchServiceJobSchedulerTestSuite) TestProcess_RejectExtraExecutions() {
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(2),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBidAccepted, 0),
		WithPartitionedExecution("node1", models.ExecutionStateAskForBidAccepted, 0), // Same partition as first one
		WithPartitionedExecution("node2", models.ExecutionStateBidAccepted, 1),       // Different partition
	)
	scenario.executions[1].ModifyTime = scenario.executions[0].ModifyTime + 1 // trick scheduler to reject the second execution
	s.mockJobStore(scenario)

	// mock active executions' nodes to be healthy
	s.mockAllNodes("node0", "node1", "node2")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         scenario.evaluation,
		JobState:           models.JobStateTypeRunning,
		ApprovedExecutions: []string{scenario.executions[0].ID},
		StoppedExecutions:  []string{scenario.executions[1].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// TestProcess_NotEnoughNodes_NoQueueing tests the case where there are not enough nodes to run the job
// and queueing not enabled in the job spec.
// The scheduling should fail
func (s *BatchServiceJobSchedulerTestSuite) TestProcess_NotEnoughNodes_NoQueueing() {
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(3),
		WithCreateTime(s.clock.Now().Add(-1*time.Second).UnixNano()), // created in the past to avoid queueing
	)
	s.mockJobStore(scenario)

	// we need 3 executions. discover fewer nodes
	s.mockMatchingNodes(scenario, "node0", "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		JobState:   models.JobStateTypeFailed,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// TestProcess_NotEnoughNodes_Queue tests the case where there are not enough nodes to run the job
// and queueing is enabled in the job spec.
// The scheduling should queue the job, and the few nodes that were discovered should be asked to bid
func (s *BatchServiceJobSchedulerTestSuite) TestProcess_NotEnoughNodes_Queue() {
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(3),
		WithCreateTime(s.clock.Now().UnixNano()),
		WithQueueTimeout(60*time.Minute),
	)
	s.mockJobStore(scenario)

	// we need 3 executions. discover fewer nodes
	s.mockMatchingNodes(scenario, "node0", "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node0", PartitionIndex: 0},
			{NodeID: "node1", PartitionIndex: 1},
		},
		ExpectedNewEvaluations: []ExpectedEvaluation{
			{
				TriggeredBy: models.EvalTriggerJobQueue,
				WaitUntil:   s.clock.Now().Add(s.scheduler.queueBackoff),
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

// TestProcess_NotEnoughNodes_QueueWithRunning tests the case where there are not enough nodes to run the job
// and queueing is enabled in the job spec.
// The scheduling should queue the job with some nodes running subset of the executions.
func (s *BatchServiceJobSchedulerTestSuite) TestProcess_NotEnoughNodes_QueueWithRunning() {
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(3),
		WithCreateTime(s.clock.Now().UnixNano()),
		WithQueueTimeout(60*time.Minute),
		WithPartitionedExecution("node0", models.ExecutionStateAskForBidAccepted, 0),
	)
	s.mockJobStore(scenario)

	// make sure existing node0 is healthy
	s.mockAllNodes("node0", "node1")
	// only discover one node for the remaining 2 executions
	s.mockMatchingNodes(scenario, "node1")

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         scenario.evaluation,
		JobState:           models.JobStateTypeRunning,
		ApprovedExecutions: []string{scenario.executions[0].ID},
		NewExecutions: []*models.Execution{
			{NodeID: "node1", PartitionIndex: 1},
		},
		ExpectedNewEvaluations: []ExpectedEvaluation{
			{
				TriggeredBy: models.EvalTriggerJobQueue,
				WaitUntil:   s.clock.Now().Add(s.scheduler.queueBackoff),
			},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}

func (s *BatchServiceJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	terminalStates := []models.JobStateType{
		models.JobStateTypeStopped,
		models.JobStateTypeCompleted,
		models.JobStateTypeFailed,
	}

	for _, terminalState := range terminalStates {
		s.T().Run(terminalState.String(), func(t *testing.T) {
			scenario := NewScenario(
				WithJobType(s.jobType),
				WithCount(3),
				WithJobState(terminalState),
				WithPartitionedExecution("node0", models.ExecutionStateAskForBid, 0),
				WithPartitionedExecution("node1", models.ExecutionStateBidAccepted, 1),
				WithPartitionedExecution("node2", models.ExecutionStateCompleted, 2),
			)
			s.mockJobStore(scenario)

			matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
				Evaluation: scenario.evaluation,
				StoppedExecutions: []string{
					scenario.executions[0].ID,
					scenario.executions[1].ID},
			})
			s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
			s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
		})
	}
}

func (s *BatchServiceJobSchedulerTestSuite) TestProcess_ShouldPreservePartitionOnRetry() {
	// Test that failed partition is retried and maintains same index
	scenario := NewScenario(
		WithJobType(s.jobType),
		WithCount(3),
		WithPartitionedExecution("node0", models.ExecutionStateBidAccepted, 0), // partition 0 running
		WithPartitionedExecution("node1", models.ExecutionStateFailed, 1),      // partition 1 failed
		WithPartitionedExecution("node2", models.ExecutionStateBidAccepted, 2), // partition 2 running
	)

	s.mockJobStore(scenario)
	s.mockAllNodes("node0", "node1", "node2", "node3")
	s.mockMatchingNodes(scenario, "node3") // New node for retry

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: scenario.evaluation,
		NewExecutions: []*models.Execution{
			{NodeID: "node3", PartitionIndex: 1},
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(context.Background(), scenario.evaluation))
}
