//go:build unit || !integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	job, jobState, evaluation := mockJob()
	jobState.Executions = []model.ExecutionState{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// we need 3 executions. discover enough nodes
	nodeInfos := []model.NodeInfo{
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
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[execBidAccepted].NodeID),
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
	job, jobState, evaluation := mockJob()

	// mock active executions to be in pending approval state
	job.Spec.Deal.Concurrency = 2
	jobState.Executions[0].State = model.ExecutionStateAskForBidAccepted                       // pending approval
	jobState.Executions[1].State = model.ExecutionStateAskForBidAccepted                       // pending approval
	jobState.Executions[2].State = model.ExecutionStateBidAccepted                             // already running
	jobState.Executions[1].UpdateTime = jobState.Executions[0].UpdateTime.Add(1 * time.Second) // trick scheduler to reject the second execution
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[0].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[1].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[2].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		ApprovedExecutions: []model.ExecutionID{jobState.Executions[0].ID()},
		StoppedExecutions:  []model.ExecutionID{jobState.Executions[1].ID()},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_TooManyExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	job.Spec.Deal.Concurrency = 2
	jobState.Executions[execBidAccepted].Version = jobState.Executions[execAskForBid].Version + 1
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock active executions' nodes to be healthy
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[execBidAccepted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:        evaluation,
		StoppedExecutions: []model.ExecutionID{jobState.Executions[execAskForBid].ID()},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcessFail_NotEnoughExecutions() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	jobState.Executions = []model.ExecutionState{} // no executions yet
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// we need 3 executions. discover fewer nodes
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), nodeIDs[0]),
		*mockNodeInfo(s.T(), nodeIDs[1]),
	}
	s.mockNodeSelection(job, nodeInfos)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateError,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_WhenJobIsStopped_ShouldMarkNonTerminalExecutionsAsStopped() {
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
			s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
			s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

			matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
				Evaluation: evaluation,
				StoppedExecutions: []model.ExecutionID{
					jobState.Executions[execAskForBid].ID(),
					jobState.Executions[execBidAccepted].ID(),
				},
			})
			s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
			s.Require().NoError(s.scheduler.Process(ctx, evaluation))
		})
	}
}

func (s *BatchJobSchedulerTestSuite) TestFailUnhealthyExecs_ShouldMarkExecutionsOnUnhealthyNodesAsFailed() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mock node discoverer to exclude the node in BidAccepted state
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[execAskForBid].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[execCanceled].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)
	s.mockNodeSelection(job, nodeInfos)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation:         evaluation,
		NewExecutionsNodes: []peer.ID{nodeInfos[1].PeerInfo.ID},
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[execBidAccepted].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsCompleted() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	jobState.Executions[execAskForBid].State = model.ExecutionStateCompleted
	jobState.Executions[execBidAccepted].State = model.ExecutionStateCompleted
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateCompleted,
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_OnMoreNodes() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)

	// mark all nodes as unhealthy so that we don't retry on other nodes
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return([]model.NodeInfo{}, nil)
	s.mockNodeSelection(job, []model.NodeInfo{})

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateError,
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[execAskForBid].ID(),
			jobState.Executions[execBidAccepted].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) TestProcess_ShouldMarkJobAsFailed_OnNoRetry() {
	ctx := context.Background()
	job, jobState, evaluation := mockJob()
	s.jobStore.EXPECT().GetJob(gomock.Any(), job.ID()).Return(*job, nil)
	s.jobStore.EXPECT().GetJobState(gomock.Any(), job.ID()).Return(*jobState, nil)
	s.scheduler.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false})

	// mark askForBid exec as lost so we attempt to retry
	nodeInfos := []model.NodeInfo{
		*mockNodeInfo(s.T(), jobState.Executions[execBidAccepted].NodeID),
		*mockNodeInfo(s.T(), jobState.Executions[execCompleted].NodeID),
	}
	s.nodeDiscoverer.EXPECT().ListNodes(gomock.Any()).Return(nodeInfos, nil)

	matcher := NewPlanMatcher(s.T(), PlanMatcherParams{
		Evaluation: evaluation,
		JobState:   model.JobStateError,
		StoppedExecutions: []model.ExecutionID{
			jobState.Executions[execAskForBid].ID(),
			jobState.Executions[execBidAccepted].ID(),
		},
	})
	s.planner.EXPECT().Process(gomock.Any(), matcher).Times(1)
	s.Require().NoError(s.scheduler.Process(ctx, evaluation))
}

func (s *BatchJobSchedulerTestSuite) mockNodeSelection(job *model.Job, nodeInfos []model.NodeInfo) {
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

func mockJob() (*model.Job, *model.JobState, *models.Evaluation) {
	job := mock.Job()
	job.Spec.Deal.Concurrency = 3

	jobState := mock.JobState(job.ID(), 5)
	for i := 0; i < 5; i++ {
		jobState.Executions[i].NodeID = nodeIDs[i]
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
