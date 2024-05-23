//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

const (
	execAskForBid   = 0
	execBidAccepted = 1
	execCompleted   = 2
	execCanceled    = 3
	execFailed      = 4
)

var nodeIDs = []string{
	"Node0",
	"Node1",
	"Node2",
	"Node3",
	"Node4",
}

type BatchJobSchedulerTestSuite struct {
	suite.Suite
	clock         *clock.Mock
	jobStore      *jobstore.MockStore
	planner       *orchestrator.MockPlanner
	nodeSelector  *orchestrator.MockNodeSelector
	retryStrategy orchestrator.RetryStrategy
	scheduler     *BatchServiceJobScheduler
}

func (s *BatchJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.clock = clock.NewMock()
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeSelector = orchestrator.NewMockNodeSelector(ctrl)
	s.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true})

	s.scheduler = NewBatchServiceJobScheduler(BatchServiceJobSchedulerParams{
		JobStore:      s.jobStore,
		Planner:       s.planner,
		NodeSelector:  s.nodeSelector,
		RetryStrategy: s.retryStrategy,
		Clock:         s.clock,
	})

	// we only want to freeze time to have more deterministic tests.
	// It doesn't matter what time it is as we are using relative time to this value
	s.clock.Set(time.Now())
}

func TestBatchSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchJobSchedulerTestSuite))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldCreateEnoughExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockJob()
	executions := []models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// we need 3 executions. discover enough nodes
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), nodeIDs[0]),
		*fakeNodeInfo(s.T(), nodeIDs[1]),
		*fakeNodeInfo(s.T(), nodeIDs[2]),
		*fakeNodeInfo(s.T(), nodeIDs[3]),
		*fakeNodeInfo(s.T(), nodeIDs[4]),
	}
	s.mockNodeSelection(job, nodeInfos, job.Count)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		NewExecutionsNodes: []string{
			nodeInfos[0].ID(),
			nodeInfos[1].ID(),
			nodeInfos[2].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execBidAccepted].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	// empty plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_RejectExtraExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()

	// mock active executions to be in pending approval state
	job.Count = 2
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateAskForBidAccepted) // pending approval
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateAskForBidAccepted) // pending approval
	executions[2].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)       // already running
	executions[1].ModifyTime = executions[0].ModifyTime + 1                                       // trick scheduler to reject the second execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[0].NodeID),
		*fakeNodeInfo(s.T(), executions[1].NodeID),
		*fakeNodeInfo(s.T(), executions[2].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		JobState:           models.JobStateTypeRunning,
		ApprovedExecutions: []string{executions[0].ID},
		StoppedExecutions:  []string{executions[1].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_TooManyExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	job.Count = 2
	executions[execBidAccepted].Revision = executions[execAskForBid].Revision + 1
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execBidAccepted].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        evaluation,
		StoppedExecutions: []string{executions[execAskForBid].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcessFail_NotEnoughExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockJob()
	executions := []models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// we need 3 executions. discover fewer nodes
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), nodeIDs[0]),
		*fakeNodeInfo(s.T(), nodeIDs[1]),
	}
	s.mockNodeSelection(job, nodeInfos, job.Count)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	terminalStates := []models.JobStateType{
		models.JobStateTypeStopped,
		models.JobStateTypeCompleted,
		models.JobStateTypeFailed,
	}

	for _, terminalState := range terminalStates {
		s.T().Run(terminalState.String(), func(t *testing.T) {
			ctx := context.Background()
			job, executions, evaluation := mockJob()
			job.State = models.NewJobState(terminalState)
			s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
			s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

			matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
				Evaluation: evaluation,
				StoppedExecutions: []string{
					executions[execAskForBid].ID,
					executions[execBidAccepted].ID,
				},
			})
			s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
			s.Require().NoError(s.scheduler.Process(ctx, evaluation))
		})
	}
}

func (s *BatchJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the node in BidAccepted state
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execCanceled].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos, 1)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []string{nodeInfos[0].ID()},
		StoppedExecutions: []string{
			executions[execBidAccepted].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	executions[execAskForBid].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
	executions[execBidAccepted].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoMoreNodes() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return([]models.NodeInfo{}, nil)
	s.mockNodeSelection(job, []models.NodeInfo{}, job.Count-1) // exclude completed exec.

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			executions[execAskForBid].ID,
			executions[execBidAccepted].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoRetry() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execBidAccepted].NodeID),
		*fakeNodeInfo(s.T(), executions[execCompleted].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			executions[execAskForBid].ID,
			executions[execBidAccepted].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldStopExpiredExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	// Set the job type to batch and the timeout to 60 minutes
	job.Task().Timeouts.ExecutionTimeout = int64((60 * time.Minute).Seconds())
	// Set the start time of the executions to exceed the timeout
	for i := range executions {
		executions[i].ModifyTime = s.clock.Now().Add(-90 * time.Minute).UnixNano()
	}
	// override the first execution to be running as well to time it out
	executions[execAskForBid].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)

	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execBidAccepted].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos, 2)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []string{nodeInfos[0].ID(), nodeInfos[1].ID()},
		StoppedExecutions: []string{
			executions[execAskForBid].ID,
			executions[execBidAccepted].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) mockNodeSelection(job *models.Job, nodeInfos []models.NodeInfo, desiredCount int) {
	if len(nodeInfos) < desiredCount {
		s.nodeSelector.EXPECT().TopMatchingNodes(gomock.Any(), job, desiredCount).Return(nil, orchestrator.ErrNotEnoughNodes{})
	} else {
		s.nodeSelector.EXPECT().TopMatchingNodes(
			gomock.Any(),
			job,
			desiredCount,
		).Return(nodeInfos, nil)
	}
}

func mockJob() (*models.Job, []models.Execution, *models.Evaluation) {
	job := mock.Job()
	job.Type = models.JobTypeBatch
	job.Count = 3

	executionCount := 5
	executions := make([]models.Execution, executionCount)
	for i, e := range mock.Executions(job, executionCount) {
		e.NodeID = nodeIDs[i]
		executions[i] = *e
	}
	executions[execAskForBid].ComputeState = models.NewExecutionState(models.ExecutionStateAskForBid)
	executions[execBidAccepted].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[execCompleted].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
	executions[execCanceled].ComputeState = models.NewExecutionState(models.ExecutionStateCancelled)
	executions[execFailed].ComputeState = models.NewExecutionState(models.ExecutionStateFailed)

	evaluation := &models.Evaluation{
		JobID: job.ID,
		ID:    uuid.NewString(),
	}
	return job, executions, evaluation
}
