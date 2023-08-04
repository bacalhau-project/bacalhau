//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type OpsJobSchedulerTestSuite struct {
	suite.Suite
	jobStore       *jobstore.MockStore
	planner        *orchestrator.MockPlanner
	nodeDiscoverer *orchestrator.MockNodeDiscoverer
	nodeRanker     *orchestrator.MockNodeRanker
	scheduler      *OpsJobScheduler
}

func (s *OpsJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeDiscoverer = orchestrator.NewMockNodeDiscoverer(ctrl)
	s.nodeRanker = orchestrator.NewMockNodeRanker(ctrl)

	s.scheduler = NewOpsJobScheduler(OpsJobSchedulerParams{
		JobStore:       s.jobStore,
		Planner:        s.planner,
		NodeDiscoverer: s.nodeDiscoverer,
		NodeRanker:     s.nodeRanker,
	})
}

func TestOpsJobSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(OpsJobSchedulerTestSuite))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldCreateNewExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.Executions = []model.ExecutionState{}
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), nodeIDs[0]),
		*mockNodeInfo(s.T(), nodeIDs[1]),
		*mockNodeInfo(s.T(), nodeIDs[2]),
	}
	s.mockNodeSelection(job, nodeInfos)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:               evaluation,
		NewExecutionDesiredState: model.ExecutionDesiredStateRunning,
		NewExecutionsNodes: []peer.ID{
			nodeInfos[0].PeerInfo.ID,
			nodeInfos[1].PeerInfo.ID,
			nodeInfos[2].PeerInfo.ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.Executions[0].State = model.ExecutionStateCompleted // Simulate a completed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkLostExecutionsOnUnhealthyNodes() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.Executions[0].State = model.ExecutionStateBidAccepted
	jobState.Executions[1].State = model.ExecutionStateBidAccepted
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[1].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []peer.ID{},
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[0].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.Executions[0].State = model.ExecutionStateBidAccepted // running, but lost node
	jobState.Executions[1].State = model.ExecutionStateFailed      // failed execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock node discoverer to exclude the first node
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[1].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateError,
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[0].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.State = model.JobStateCancelled // Simulate a canceled job
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[0].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) TestProcessFail_NoMatchingNodes() {
	ctx := context.Background()
	job, jobState, evaluation := mockOpsJob()
	jobState.Executions = []model.ExecutionState{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)
	s.mockNodeSelection(job, []model.NodeInfo{})

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateError,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *OpsJobSchedulerTestSuite) mockNodeSelection(job *model.Job, nodeInfos []model.NodeInfo) {
	s.nodeDiscoverer.EXPECT().FindNodes(gomock.Any(), *job).Return(nodeInfos, nil)

	nodeRanks := make([]orchestrator.NodeRank, len(nodeInfos))
	for i, nodeInfo := range nodeInfos {
		nodeRanks[i] = orchestrator.NodeRank{
			NodeInfo: nodeInfo,
			Rank:     i,
		}
	}
	s.nodeRanker.EXPECT().RankNodes(gomock.Any(), *job, nodeInfos).Return(nodeRanks, nil)
}

func mockOpsJob() (*model.Job, *model.JobState, *models.Evaluation) {
	job := mock.Job()
	jobState := mock.JobState(job.ID(), 2)
	for i := 0; i < 2; i++ {
		jobState.Executions[i].NodeID = nodeIDs[i]
	}
	jobState.Executions[0].State = model.ExecutionStateBidAccepted
	jobState.Executions[1].State = model.ExecutionStateCompleted

	evaluation := &models.Evaluation{
		JobID: job.ID(),
		Type:  model.JobTypeOps,
		ID:    uuid.NewString(),
	}

	return job, jobState, evaluation
}
