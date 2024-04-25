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
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type DaemonJobSchedulerTestSuite struct {
	suite.Suite
	jobStore     *jobstore.MockStore
	planner      *orchestrator.MockPlanner
	nodeSelector *orchestrator.MockNodeSelector
	scheduler    *DaemonJobScheduler
}

func (s *DaemonJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeSelector = orchestrator.NewMockNodeSelector(ctrl)

	s.scheduler = NewDaemonJobScheduler(
		s.jobStore,
		s.planner,
		s.nodeSelector,
		orchestrator.NodeSelectionConstraints{
			RequireConnected: true,
			RequireApproval:  true,
		},
	)
}

func TestDaemonJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(DaemonJobSchedulerTestSuite))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldCreateNewExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockDaemonJob()
	executions := []models.Execution{}
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), nodeIDs[0]),
		*fakeNodeInfo(s.T(), nodeIDs[1]),
		*fakeNodeInfo(s.T(), nodeIDs[2]),
	}
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), gomock.Any(), gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:               evaluation,
		JobState:                 models.JobStateTypeRunning,
		NewExecutionDesiredState: models.ExecutionDesiredStateRunning,
		NewExecutionsNodes: []string{
			nodeInfos[0].ID(),
			nodeInfos[1].ID(),
			nodeInfos[2].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

// It is a bug if a long running execution is completed. The scheduler should just ignore it
// and NOT mark the job as completed
func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldNOTMarkJobAsCompleted() {
	ctx := context.Background()
	job, executions, evaluation := mockDaemonJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted) // Simulate a completed execution
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted) // Simulate a completed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), gomock.Any(), gomock.Any()).Return([]models.NodeInfo{}, nil)

	// Noop plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldMarkLostExecutionsOnUnhealthyNodes() {
	ctx := context.Background()
	job, executions, evaluation := mockDaemonJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[1].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), job, gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []string{},
		StoppedExecutions: []string{
			executions[0].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

// Even when an execution has failed, we don't mark the job as failed and continue waiting
// for more nodes that match the job selection to join.
// This requires a revisit in the future if all or a high percentage of nodes keep failing
func (s *DaemonJobSchedulerTestSuite) TestProcess_ShouldNOTMarkJobAsFailed() {
	ctx := context.Background()
	job, executions, evaluation := mockDaemonJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted) // running, but lost node
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateFailed)      // failed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[1].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), job, gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		StoppedExecutions: []string{
			executions[0].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	ctx := context.Background()
	job, executions, evaluation := mockDaemonJob()
	job.State = models.NewJobState(models.JobStateTypeStopped) // Simulate a canceled job
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		StoppedExecutions: []string{
			executions[0].ID,
			executions[1].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *DaemonJobSchedulerTestSuite) TestProcessFail_NoMatchingNodes() {
	ctx := context.Background()
	job, _, evaluation := mockDaemonJob()
	executions := []models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), job, gomock.Any()).Return([]models.NodeInfo{}, nil)

	// Noop plan
	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func mockDaemonJob() (*models.Job, []models.Execution, *models.Evaluation) {
	job := mock.Job()

	executionCount := 2
	executions := make([]models.Execution, executionCount)
	for i, e := range mock.Executions(job, executionCount) {
		e.NodeID = nodeIDs[i]
		executions[i] = *e
	}

	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateAskForBidAccepted)

	evaluation := &models.Evaluation{
		JobID: job.ID,
		Type:  models.JobTypeDaemon,
		ID:    uuid.NewString(),
	}

	return job, executions, evaluation
}
