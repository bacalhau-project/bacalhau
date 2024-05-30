//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

type OverSubscriptionNodeRankerSuite struct {
	suite.Suite
}

func (suite *OverSubscriptionNodeRankerSuite) TestRankNodes() {
	testCases := []struct {
		name     string
		factor   float64
		node     models.NodeInfo
		expected int
	}{
		{
			name:     "nil ComputeNodeInfo",
			factor:   0.5,
			node:     models.NodeInfo{ComputeNodeInfo: nil},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "empty ComputeNodeInfo",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{},
			},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "high queue usage",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					QueueUsedCapacity: models.Resources{CPU: 10, Memory: 40000, Disk: 200000, GPU: 5},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "queue usage equal to factor times total capacity",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 4},
				},
			},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "queue usage slightly less than factor times total capacity",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 1.99, Memory: 7999, Disk: 49999, GPU: 3},
				},
			},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "queue usage slightly more than factor times total capacity",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2.01, Memory: 8001, Disk: 50001, GPU: 5},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only CPU over-subscribed",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2.01, Memory: 8000, Disk: 50000, GPU: 4},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only Memory over-subscribed",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2, Memory: 8001, Disk: 50000, GPU: 4},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only Disk over-subscribed",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50001, GPU: 4},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only GPU over-subscribed",
			factor: 0.5,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 5},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "zero factor and non-empty queue",
			factor: 0.0,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					QueueUsedCapacity: models.Resources{CPU: 0.01, Memory: 10, Disk: 100, GPU: 0},
				},
			},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "zero factor and empty queue",
			factor: 0.0,
			node: models.NodeInfo{
				ComputeNodeInfo: &models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					QueueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			expected: orchestrator.RankPossible,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Set up the ranker with the factor for this test case
			ranker, err := NewOverSubscriptionNodeRanker(tc.factor)
			suite.NoError(err)

			job := models.Job{} // Mock job
			ranks, err := ranker.RankNodes(context.Background(), job, []models.NodeInfo{tc.node})
			suite.NoError(err)
			suite.Equal(tc.expected, ranks[0].Rank)
		})
	}
}

// TestInvalidFactor tests that the ranker returns an error when the factor is invalid.
func (suite *OverSubscriptionNodeRankerSuite) TestInvalidFactor() {
	_, err := NewOverSubscriptionNodeRanker(-1.0)
	suite.Error(err)
}

func TestOverSubscriptionNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(OverSubscriptionNodeRankerSuite))
}
