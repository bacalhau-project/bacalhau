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

type OpsJobSchedulerTestSuite struct {
	suite.Suite
	jobStore     *jobstore.MockStore
	planner      *orchestrator.MockPlanner
	nodeSelector *orchestrator.MockNodeSelector
	scheduler    *OpsJobScheduler
}

func (s *OpsJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeSelector = orchestrator.NewMockNodeSelector(ctrl)

	s.scheduler = NewOpsJobScheduler(
		s.jobStore,
		s.planner,
		s.nodeSelector,
		orchestrator.NodeSelectionConstraints{
			RequireConnected: false,
			RequireApproval:  false,
		},
	)
}

func TestOpsJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(OpsJobSchedulerTestSuite))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldCreateNewExecutions() {
	ctx := context.Background()
	job, _, evaluation := mockOpsJob()
	executions := []models.Execution{}
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), nodeIDs[0]),
		*fakeNodeInfo(s.T(), nodeIDs[1]),
		*fakeNodeInfo(s.T(), nodeIDs[2]),
	}
	s.mockNodeSelection(job, nodeInfos)

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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	ctx := context.Background()
	job, executions, evaluation := mockOpsJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted) // Simulate a completed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkLostExecutionsOnUnhealthyNodes() {
	ctx := context.Background()
	job, executions, evaluation := mockOpsJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[1].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

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

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed() {
	ctx := context.Background()
	job, executions, evaluation := mockOpsJob()
	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted) // running, but lost node
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateFailed)      // failed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []models.NodeInfo{
		*fakeNodeInfo(s.T(), executions[1].NodeID),
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
		StoppedExecutions: []string{
			executions[0].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	ctx := context.Background()
	job, executions, evaluation := mockOpsJob()
	job.State = models.NewJobState(models.JobStateTypeStopped) // Simulate a canceled job
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		StoppedExecutions: []string{
			executions[0].ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcessFail_NoMatchingNodes() {
	ctx := context.Background()
	job, _, evaluation := mockOpsJob()
	executions := []models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), jobstore.GetExecutionsOptions{JobID: job.ID}).Return(executions, nil)
	s.mockNodeSelection(job, []models.NodeInfo{})

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeFailed,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) mockNodeSelection(job *models.Job, nodeInfos []models.NodeInfo) {
	s.nodeSelector.EXPECT().AllMatchingNodes(gomock.Any(), job, gomock.Any()).Return(nodeInfos, nil)
}

func mockOpsJob() (*models.Job, []models.Execution, *models.Evaluation) {
	job := mock.Job()

	executionCount := 2
	executions := make([]models.Execution, executionCount)
	for i, e := range mock.Executions(job, executionCount) {
		e.NodeID = nodeIDs[i]
		executions[i] = *e
	}

	executions[0].ComputeState = models.NewExecutionState(models.ExecutionStateBidAccepted)
	executions[1].ComputeState = models.NewExecutionState(models.ExecutionStateCompleted)

	evaluation := &models.Evaluation{
		JobID: job.ID,
		Type:  models.JobTypeOps,
		ID:    uuid.NewString(),
	}

	return job, executions, evaluation
}
