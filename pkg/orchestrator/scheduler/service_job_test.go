//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"

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
	execServiceAskForBid    = 0
	execServiceBidAccepted1 = 1
	execServiceBidAccepted2 = 2
	execServiceCanceled     = 3
	execServiceFailed       = 4
)

type ServiceJobSchedulerTestSuite struct {
	suite.Suite
	jobStore      *jobstore.MockStore
	planner       *orchestrator.MockPlanner
	nodeSelector  *orchestrator.MockNodeSelector
	retryStrategy orchestrator.RetryStrategy
	scheduler     *BatchServiceJobScheduler
}

func (s *ServiceJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeSelector = orchestrator.NewMockNodeSelector(ctrl)
	s.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true})

	s.scheduler = NewBatchServiceJobScheduler(
		s.jobStore,
		s.planner,
		s.nodeSelector,
		s.retryStrategy,
	)
}

func TestServiceSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceJobSchedulerTestSuite))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_ShouldCreateEnoughExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockServiceJob()
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

func (s *ServiceJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execServiceAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted1].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted2].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	// empty plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_RejectExtraExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()

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

func (s *ServiceJobSchedulerTestSuite) TestProcess_TooManyExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	job.Count = 2
	executions[execServiceBidAccepted1].Revision = executions[execServiceAskForBid].Revision + 1
	executions[execServiceBidAccepted2].Revision = executions[execServiceAskForBid].Revision + 1
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execServiceAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted1].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted2].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        evaluation,
		StoppedExecutions: []string{executions[execServiceAskForBid].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcessFail_NotEnoughExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockServiceJob()
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

func (s *ServiceJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	terminalStates := []models.JobStateType{
		models.JobStateTypeStopped,
		models.JobStateTypeCompleted,
		models.JobStateTypeFailed,
	}

	for _, terminalState := range terminalStates {
		s.T().Run(terminalState.String(), func(t *testing.T) {
			ctx := context.Background()
			job, executions, evaluation := mockServiceJob()
			job.State = models.NewJobState(terminalState)
			s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
			s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

			matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
				Evaluation: evaluation,
				StoppedExecutions: []string{
					executions[execServiceAskForBid].ID,
					executions[execServiceBidAccepted1].ID,
					executions[execServiceBidAccepted2].ID,
				},
			})
			s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
			s.Require().NoError(s.scheduler.Process(ctx, evaluation))
		})
	}
}

func (s *ServiceJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the node in BidAccepted state
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execServiceAskForBid].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceCanceled].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos, 2)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		NewExecutionsNodes: []string{
			nodeInfos[0].ID(),
			nodeInfos[1].ID(),
		},
		StoppedExecutions: []string{
			executions[execServiceBidAccepted1].ID,
			executions[execServiceBidAccepted2].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

// It is a bug if a long running execution is completed. The scheduler treat those as failed executions,
// try to reschedule, or fail the job if can no longer reschedule
func (s *ServiceJobSchedulerTestSuite) TestProcess_TreatCompletedExecutionsAsFailed() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	executions[execServiceAskForBid].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
	executions[execServiceBidAccepted1].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// discover all nodes to avoid treating active executions as unhealthy
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), nodeIDs[0]),
		*fakeNodeInfo(s.T(), nodeIDs[1]),
		*fakeNodeInfo(s.T(), nodeIDs[2]),
		*fakeNodeInfo(s.T(), nodeIDs[3]),
		*fakeNodeInfo(s.T(), nodeIDs[4]),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos, 2)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		NewExecutionsNodes: []string{
			nodeInfos[0].ID(),
			nodeInfos[1].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoMoreNodes() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return([]models.NodeInfo{}, nil)
	s.mockNodeSelection(job, []models.NodeInfo{}, job.Count)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			executions[execServiceAskForBid].ID,
			executions[execServiceBidAccepted1].ID,
			executions[execServiceBidAccepted2].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *ServiceJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_NoRetry() {
	ctx := context.Background()
	job, executions, evaluation := mockServiceJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted1].NodeID),
		*fakeNodeInfo(s.T(), executions[execServiceBidAccepted2].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			executions[execServiceAskForBid].ID,
			executions[execServiceBidAccepted1].ID,
			executions[execServiceBidAccepted2].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *ServiceJobSchedulerTestSuite) mockNodeSelection(job *models.Job, nodeInfos []models.NodeInfo, desiredCount int) {
	if len(nodeInfos) < desiredCount {
		s.nodeSelector.EXPECT().TopMatchingNodes(gomock.Any(), job, desiredCount).Return(nil, orchestrator.ErrNotEnoughNodes{})
	} else {
		s.nodeSelector.EXPECT().TopMatchingNodes(gomock.Any(), job, desiredCount).Return(nodeInfos, nil)
	}
}

func mockServiceJob() (*models.Job, []models.Execution, *models.Evaluation) {
	job := mock.Job()
	job.Type = models.JobTypeService
	job.Count = 3

	executionCount := 5
	executions := make([]models.Execution, executionCount)
	for i, e := range mock.Executions(job, executionCount) {
		e.NodeID = nodeIDs[i]
		executions[i] = *e
	}
	executions[execServiceAskForBid].ComputeState = models.NewExecutionState(models.ExecutionStateAskForBid)
	executions[execServiceBidAccepted1].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[execServiceBidAccepted2].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[execServiceCanceled].ComputeState = models.NewExecutionState(models.ExecutionStateCancelled)
	executions[execServiceFailed].ComputeState = models.NewExecutionState(models.ExecutionStateFailed)

	evaluation := &models.Evaluation{
		JobID: job.ID,
		ID:    uuid.NewString(),
	}
	return job, executions, evaluation
}
