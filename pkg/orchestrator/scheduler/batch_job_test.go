//go:build unit || !integration

package scheduler

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	execAskForBid   = 0
	execBidAccepted = 1
	execCompleted   = 2
	execCanceled    = 3
	execFailed      = 4
)

type SchedulerTestSuite struct {
	suite.Suite
	jobStore       *jobstore.MockStore
	planner        *orchestrator.MockPlanner
	nodeDiscoverer *requester.MockNodeDiscoverer
	nodeRanker     *requester.MockNodeRanker
	scheduler      *BatchJobScheduler
}

func (s *SchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeDiscoverer = requester.NewMockNodeDiscoverer(ctrl)
	s.nodeRanker = requester.NewMockNodeRanker(ctrl)

	s.scheduler = NewBatchJobScheduler(&BatchJobSchedulerParams{
		JobStore:       s.jobStore,
		Planner:        s.planner,
		NodeDiscoverer: s.nodeDiscoverer,
		NodeRanker:     s.nodeRanker,
	})
}

func TestSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(SchedulerTestSuite))
}

func (s *SchedulerTestSuite) TestProcess_ShouldCreateEnoughExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	jobState.Executions = []model.ExecutionState{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// we need 3 executions. discover enough nodes
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo("node-1"),
		*mockNodeInfo("node-2"),
		*mockNodeInfo("node-3"),
		*mockNodeInfo("node-4"),
		*mockNodeInfo("node-5"),
	}
	s.mockNodeSelection(job, nodeInfos)

	s.planner.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, plan *models.Plan) {
		assertPlan(s.T(), plan, evaluation.ID, 3, 0)
		// should've selected the nodes with higher rank
		s.Equal(nodeInfos[4].PeerInfo.ID.String(), plan.NewExecutions[0].NodeID)
		s.Equal(nodeInfos[3].PeerInfo.ID.String(), plan.NewExecutions[1].NodeID)
		s.Equal(nodeInfos[2].PeerInfo.ID.String(), plan.NewExecutions[2].NodeID)
	})
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *SchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(jobState.Executions[execBidAccepted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	// empty plan
	s.planner.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, plan *models.Plan) {
		assertPlan(s.T(), plan, evaluation.ID, 0, 0)
	})
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *SchedulerTestSuite) TestProcess_TooManyExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	job.Spec.Deal.Concurrency = 2
	jobState.Executions[execBidAccepted].Version = jobState.Executions[execAskForBid].Version + 1
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(jobState.Executions[execBidAccepted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	s.planner.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, plan *models.Plan) {
		assertPlan(s.T(), plan, evaluation.ID, 0, 1)
		assertPlanStoppedExecution(s.T(), plan, 0, jobState.Executions[execAskForBid].ID(), model.ExecutionStateCancelled)
	})
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *SchedulerTestSuite) TestProcessFail_NotEnoughExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	jobState.Executions = []model.ExecutionState{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// we need 3 executions. discover enough nodes
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo("node-1"),
		*mockNodeInfo("node-2"),
	}
	s.mockNodeSelection(job, nodeInfos)
	s.Require().Error(s.scheduler.Process(ctx, evaluation))
}

func (s *SchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
	terminalStates := []model.JobStateType{
		model.JobStateCancelled,
		model.JobStateCompleted,
		model.JobStateError,
	}

	for _, terminalState := range terminalStates {
		s.T().Run(terminalState.String(), func(t *testing.T) {
			ctx := context.Background()
			job, jobState, evaluation := mockJob()
			jobState.State = terminalState
			s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

			s.planner.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, plan *models.Plan) {
				assertPlan(s.T(), plan, evaluation.ID, 0, 2)
				assertPlanStoppedExecution(s.T(), plan, 0, jobState.Executions[execAskForBid].ID(), model.ExecutionStateCancelled)
				assertPlanStoppedExecution(s.T(), plan, 1, jobState.Executions[execBidAccepted].ID(), model.ExecutionStateCancelled)
			})

			s.Require().NoError(s.scheduler.Process(ctx, evaluation))
		})
	}
}

func (s *SchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock node discoverer to exclude the node in BidAccepted state
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(jobState.Executions[execCanceled].NodeID),
	}
	s.mockNodeSelection(job, nodeInfos)

	s.planner.EXPECT().Process(gomock.Any(), gomock.Any()).Return(nil).Do(func(ctx context.Context, plan *models.Plan) {
		assertPlan(s.T(), plan, evaluation.ID, 1, 1)
		assertPlanStoppedExecution(s.T(), plan, 0, jobState.Executions[execBidAccepted].ID(), model.ExecutionStateFailed)
		// should've selected the node with higher rank
		s.Equal(nodeInfos[1].PeerInfo.ID.String(), plan.NewExecutions[0].NodeID)
	})

	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *SchedulerTestSuite) mockNodeSelection(job *model.Job, nodeInfos []model.NodeInfo) {
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)
	s.nodeDiscoverer.EXPECT().FindNodes(gomock.Any(), *job).Return(nodeInfos, nil)

	nodeRanks := make([]requester.NodeRank, len(nodeInfos))
	for i, nodeInfo := range nodeInfos {
		nodeRanks[i] = requester.NodeRank{
			NodeInfo: nodeInfo,
			Rank:     i,
		}
	}
	s.nodeRanker.EXPECT().RankNodes(gomock.Any(), *job, nodeInfos).Return(nodeRanks, nil)
}

func mockJob() (*model.Job, *model.JobState, *models.Evaluation) {
	job := mock.Job()
	job.Spec.Deal.Concurrency = 3

	jobState := mock.JobState(job.ID(), 5)
	for i := 0; i < 5; i++ {
		jobState.Executions[i].NodeID = fmt.Sprintf("node-%d", i)
	}
	jobState.Executions[execAskForBid].State = model.ExecutionStateAskForBid
	jobState.Executions[execBidAccepted].State = model.ExecutionStateBidAccepted
	jobState.Executions[execCompleted].State = model.ExecutionStateCompleted
	jobState.Executions[execCanceled].State = model.ExecutionStateCancelled
	jobState.Executions[execFailed].State = model.ExecutionStateFailed

	evaluation := &models.Evaluation{
		JobID: job.ID(),
		ID:    uuid.NewString(),
	}
	return job, jobState, evaluation
}

func mockNodeInfo(nodeID string) *model.NodeInfo {
	return &model.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID: peer.ID(nodeID),
		},
	}
}

func assertPlan(t *testing.T, plan *models.Plan, evalID string, newExecutionCount, stoppedExecutionCount int) {
	assert.Equal(t, evalID, plan.EvalID)
	assert.Equal(t, newExecutionCount, len(plan.NewExecutions))
	assert.Equal(t, stoppedExecutionCount, len(plan.StoppedExecutions))
}

func assertPlanStoppedExecution(t *testing.T, plan *models.Plan, index int, executionID model.ExecutionID, desiredState model.ExecutionStateType) {
	assert.Equal(t, executionID, plan.StoppedExecutions[index].ExecutionID)
	assert.Equal(t, desiredState, plan.StoppedExecutions[index].NewValues.State)
}
