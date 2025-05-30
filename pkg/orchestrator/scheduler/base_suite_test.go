//go:build unit || !integration

package scheduler

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
)

type BaseTestSuite struct {
	suite.Suite
	clock         *clock.Mock
	jobStore      *jobstore.MockStore
	planner       *orchestrator.MockPlanner
	nodeSelector  *orchestrator.MockNodeSelector
	retryStrategy orchestrator.RetryStrategy
}

func (s *BaseTestSuite) SetupTest() {
	ctrl := gomock.NewController(s.T())
	s.clock = clock.NewMock()
	s.jobStore = jobstore.NewMockStore(ctrl)
	s.planner = orchestrator.NewMockPlanner(ctrl)
	s.nodeSelector = orchestrator.NewMockNodeSelector(ctrl)
	s.retryStrategy = retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: true})

	// we only want to freeze time to have more deterministic tests.
	// It doesn't matter what time it is as we are using relative time to this value
	s.clock.Set(time.Now())
}

// mockJobStore accepts a scenario and mocks job store GetJob and GetExecutions
// to return scenario job and executions
func (s *BaseTestSuite) mockJobStore(scenario *Scenario) {
	s.jobStore.EXPECT().GetJob(gomock.Any(), scenario.job.ID).Return(*scenario.job, nil)
	s.jobStore.EXPECT().GetExecutions(
		gomock.Any(),
		jobstore.GetExecutionsOptions{
			JobID:                   scenario.job.ID,
			AllJobVersions:          true,
			CurrentLatestJobVersion: scenario.job.Version,
		},
	).Return(scenario.executions, nil)
}

func (s *BaseTestSuite) mockAllNodes(nodeIDs ...string) []models.NodeInfo {
	nodeInfos := make([]models.NodeInfo, len(nodeIDs))
	for i, nodeID := range nodeIDs {
		nodeInfos[i] = fakeNodeInfo(s.T(), nodeID)
	}
	s.nodeSelector.EXPECT().AllNodes(gomock.Any()).Return(nodeInfos, nil)
	return nodeInfos
}

func (s *BaseTestSuite) mockMatchingNodes(scenario *Scenario, nodeIDs ...string) []orchestrator.NodeRank {
	nodeRanks := make([]orchestrator.NodeRank, len(nodeIDs))
	for i, nodeID := range nodeIDs {
		nodeRanks[i] = *fakeNodeRank(s.T(), nodeID)
	}
	s.nodeSelector.EXPECT().MatchingNodes(gomock.Any(), scenario.job).Return(nodeRanks, []orchestrator.NodeRank{}, nil)
	return nodeRanks
}
