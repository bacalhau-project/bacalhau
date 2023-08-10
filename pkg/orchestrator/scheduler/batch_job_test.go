//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"
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

var nodeIDs = []string{
	"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
	"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
}

type BatchJobSchedulerTestSuite struct {
	suite.Suite
	jobStore       *jobstore.MockStore
	planner        *orchestrator.MockPlanner
	nodeDiscoverer *orchestrator.MockNodeDiscoverer
	nodeRanker     *orchestrator.MockNodeRanker
	retryStrategy  orchestrator.RetryStrategy
	scheduler      *BatchJobScheduler
}

func (s *BatchJobSchedulerTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeDiscoverer = orchestrator.NewMockNodeDiscoverer(ctrl)
	s.nodeRanker = orchestrator.NewMockNodeRanker(ctrl)
	s.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true})

	s.scheduler = NewBatchJobScheduler(BatchJobSchedulerParams{
		JobStore:       s.jobStore,
		Planner:        s.planner,
		NodeDiscoverer: s.nodeDiscoverer,
		NodeRanker:     s.nodeRanker,
		RetryStrategy:  s.retryStrategy,
	})
}

func TestSchedulerTestSuite(t *testing.T) {
	suite.Run(t, new(BatchJobSchedulerTestSuite))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldCreateEnoughExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	executions = []*models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// we need 3 executions. discover enough nodes
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), nodeIDs[0]),
		*mockNodeInfo(s.T(), nodeIDs[1]),
		*mockNodeInfo(s.T(), nodeIDs[2]),
		*mockNodeInfo(s.T(), nodeIDs[3]),
		*mockNodeInfo(s.T(), nodeIDs[4]),
	}
	s.mockNodeSelection(job, nodeInfos)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		NewExecutionsNodes: []peer.ID{
			nodeInfos[0].PeerInfo.ID,
			nodeInfos[1].PeerInfo.ID,
			nodeInfos[2].PeerInfo.ID,
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_AlreadyEnoughExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), executions[execBidAccepted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

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
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), executions[0].NodeID),
		*mockNodeInfo(s.T(), executions[1].NodeID),
		*mockNodeInfo(s.T(), executions[2].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
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
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), executions[execBidAccepted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        evaluation,
		StoppedExecutions: []string{executions[execAskForBid].ID},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcessFail_NotEnoughExecutions() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	executions = []*models.Execution{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// we need 3 executions. discover fewer nodes
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), nodeIDs[0]),
		*mockNodeInfo(s.T(), nodeIDs[1]),
	}
	s.mockNodeSelection(job, nodeInfos)

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
			s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

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
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// mock node discoverer to exclude the node in BidAccepted state
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), executions[execCanceled].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []peer.ID{nodeInfos[1].PeerInfo.ID},
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
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   models.JobStateTypeCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_OnMoreNodes() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return([]models.NodeInfo{}, nil)
	s.mockNodeSelection(job, []models.NodeInfo{})

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

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_OnNoRetry() {
	ctx := context.Background()
	job, executions, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID).Return(*job, nil)
	s.jobStore.EXPECT().GetExecutions(gomock.Any(), job.ID).Return(executions, nil)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
	nodeInfos := []models.NodeInfo{
		*mockNodeInfo(s.T(), executions[execBidAccepted].NodeID),
		*mockNodeInfo(s.T(), executions[execCompleted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

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

func (s *BatchJobSchedulerTestSuite) mockNodeSelection(job *models.Job, nodeInfos []models.NodeInfo) {
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

func mockJob() (*models.Job, []*models.Execution, *models.Evaluation) {
	job := mock.Job()
	job.Count = 3

	executions := mock.Executions(job, 5)
	for i := 0; i < 5; i++ {
		executions[i].NodeID = nodeIDs[i]
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
