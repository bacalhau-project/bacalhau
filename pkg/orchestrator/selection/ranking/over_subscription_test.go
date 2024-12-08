//go:build unit || !integration

package ranking

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type OverSubscriptionNodeRankerSuite struct {
	suite.Suite
}

func (suite *OverSubscriptionNodeRankerSuite) TestRankNodes() {
	testCases := []struct {
		name     string
		factor   float64
		node     models.NodeInfo
		job      models.Resources
		expected int
	}{
		{
			name:   "empty ComputeNodeInfo",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{},
			},
			job:      models.Resources{},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "no factor, job matches available capacity",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 4},
					QueueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			job:      models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 4},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "no factor, job slightly higher than available capacity",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 4},
					QueueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			job:      models.Resources{CPU: 2.1, Memory: 8000, Disk: 5000, GPU: 4},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "no factor, empty job, no available capacity",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					AvailableCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
					QueueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			job:      models.Resources{},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "1.5 factor, total usage matches oversubscribe capacity",
			factor: 1.5,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 1.99, Memory: 7999, Disk: 49999, GPU: 3},
				},
			},
			job:      models.Resources{CPU: 0.01, Memory: 1, Disk: 1, GPU: 1},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "1.5 factor, total usage slightly higher oversubscribe capacity",
			factor: 1.5,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 8},
					QueueUsedCapacity: models.Resources{CPU: 1.99, Memory: 7999, Disk: 49999, GPU: 3},
				},
			},
			job:      models.Resources{CPU: 0.2, Memory: 1, Disk: 1, GPU: 1},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "matches over-subscribed capacity, but empty job",
			factor: 2,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 4000, Disk: 4000, GPU: 4},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 2000, Disk: 2000, GPU: 2},
					QueueUsedCapacity: models.Resources{CPU: 6, Memory: 6000, Disk: 6000, GPU: 6},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
			expected: orchestrator.RankPossible,
		},
		{
			name:   "only CPU over-subscribed",
			factor: 2,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 4000, Disk: 4000, GPU: 4},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 2000, Disk: 2000, GPU: 2},
					QueueUsedCapacity: models.Resources{CPU: 6, Memory: 6000, Disk: 6000, GPU: 6},
				},
			},
			job:      models.Resources{CPU: 0.01, Memory: 0, Disk: 0, GPU: 0},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only Memory over-subscribed",
			factor: 2,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 4000, Disk: 4000, GPU: 4},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 2000, Disk: 2000, GPU: 2},
					QueueUsedCapacity: models.Resources{CPU: 6, Memory: 6000, Disk: 6000, GPU: 6},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 1, Disk: 0, GPU: 0},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only Disk over-subscribed",
			factor: 2,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 4000, Disk: 4000, GPU: 4},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 2000, Disk: 2000, GPU: 2},
					QueueUsedCapacity: models.Resources{CPU: 6, Memory: 6000, Disk: 6000, GPU: 6},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 0, Disk: 1, GPU: 0},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "only GPU over-subscribed",
			factor: 2,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 4000, Disk: 4000, GPU: 4},
					AvailableCapacity: models.Resources{CPU: 2, Memory: 2000, Disk: 2000, GPU: 2},
					QueueUsedCapacity: models.Resources{CPU: 6, Memory: 6000, Disk: 6000, GPU: 6},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 1},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "no factor and non-empty queue",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					AvailableCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
					QueueUsedCapacity: models.Resources{CPU: 0.01, Memory: 10, Disk: 100, GPU: 0},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
			expected: orchestrator.RankUnsuitable,
		},
		{
			name:   "no factor and empty queue",
			factor: 1,
			node: models.NodeInfo{
				ComputeNodeInfo: models.ComputeNodeInfo{
					MaxCapacity:       models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 0},
					AvailableCapacity: models.Resources{CPU: 0.01, Memory: 1, Disk: 1, GPU: 0},
					QueueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			job:      models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
			expected: orchestrator.RankPossible,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Set up the ranker with the factor for this test case
			ranker, err := NewOverSubscriptionNodeRanker(tc.factor)
			suite.NoError(err)

			job := mock.Job()
			job.Task().ResourcesConfig = &models.ResourcesConfig{
				CPU:    fmt.Sprintf("%f", tc.job.CPU),
				Memory: fmt.Sprintf("%d", tc.job.Memory),
				Disk:   fmt.Sprintf("%d", tc.job.Disk),
				GPU:    fmt.Sprintf("%d", tc.job.GPU),
			}
			ranks, err := ranker.RankNodes(context.Background(), *job, []models.NodeInfo{tc.node})
			suite.NoError(err)
			suite.Equal(tc.expected, ranks[0].Rank)
		})
	}
}

func (suite *OverSubscriptionNodeRankerSuite) TestInvalidAndValidFactors() {
	testCases := []struct {
		name   string
		factor float64
		fail   bool
	}{
		{
			name:   "invalid factor -1",
			factor: -1.0,
			fail:   true,
		},
		{
			name:   "invalid factor 0.9",
			factor: 0.9,
			fail:   true,
		},
		{
			name:   "invalid factor 0",
			factor: 0.0,
			fail:   true,
		},
		{
			name:   "valid factor 1",
			factor: 1.0,
			fail:   false,
		},
		{
			name:   "valid factor 2",
			factor: 2.0,
			fail:   false,
		},
		{
			name:   "valid factor 100",
			factor: 100.0,
			fail:   false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := NewOverSubscriptionNodeRanker(tc.factor)
			if tc.fail {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

func TestOverSubscriptionNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(OverSubscriptionNodeRankerSuite))
}
