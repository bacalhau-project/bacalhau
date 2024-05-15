//go:build unit || !integration

package ranking

import (
	"context"
	"sort"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
)

type AvailableCapacityNodeRankerSuite struct {
	suite.Suite
	ranker *AvailableCapacityNodeRanker
}

func (suite *AvailableCapacityNodeRankerSuite) SetupTest() {
	suite.ranker = NewAvailableCapacityNodeRanker()
}

type nodeScenario struct {
	nodeID            string
	availableCapacity models.Resources
	queueUsedCapacity models.Resources
}

type rankNodesTestCase struct {
	name         string
	nodes        []nodeScenario
	jobResources models.ResourcesConfig
	expected     []string
	equalCheck   bool
}

func (suite *AvailableCapacityNodeRankerSuite) TestRankNodes() {
	testCases := []rankNodesTestCase{
		{
			name: "Only available capacity",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 6, Memory: 24000, Disk: 150000, GPU: 2},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 1},
				},
			},
			expected: []string{"node2", "node1", "node3"},
		},
		{
			name: "Only queued capacity",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 40000, GPU: 1},
				},
				{
					nodeID:            "node2",
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 4000, Disk: 20000, GPU: 0},
				},
				{
					nodeID:            "node3",
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			expected: []string{"node3", "node2", "node1"},
		},
		{
			name: "Both available and queued capacities",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 40000, GPU: 1},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 6, Memory: 24000, Disk: 150000, GPU: 2},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 4000, Disk: 20000, GPU: 0},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			expected: []string{"node2", "node1", "node3"},
		},
		{
			name: "Equal capacities",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 40000, GPU: 1},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 40000, GPU: 1},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 40000, GPU: 1},
				},
			},
			equalCheck: true,
		},
		{
			name: "Zero capacities",
			nodes: []nodeScenario{
				{
					nodeID: "node1",
				},
				{
					nodeID: "node2",
				},
				{
					nodeID: "node3",
				},
			},
			equalCheck: true,
		},
		{
			name: "One node with zero capacities",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 4000, Disk: 20000, GPU: 0},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 0, Disk: 0, GPU: 0},
				},
			},
			expected: []string{"node1", "node3", "node2"},
		},
		{
			name: "High CPU weight",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 8, Memory: 16000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 4000, Disk: 50000, GPU: 0},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 4, Memory: 24000, Disk: 150000, GPU: 2},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 2000, Disk: 30000, GPU: 1},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 32000, Disk: 200000, GPU: 4},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 1000, Disk: 10000, GPU: 2},
				},
			},
			jobResources: models.ResourcesConfig{
				CPU: "1",
			},
			expected: []string{"node1", "node2", "node3"},
		},
		{
			name: "High Memory weight",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 32000, Disk: 100000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 8000, Disk: 50000, GPU: 0},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 8, Memory: 16000, Disk: 150000, GPU: 2},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 4000, Disk: 30000, GPU: 1},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 200000, GPU: 4},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 2000, Disk: 10000, GPU: 2},
				},
			},
			jobResources: models.ResourcesConfig{
				Memory: "10mb",
			},
			expected: []string{"node1", "node2", "node3"},
		},
		{
			name: "High Disk weight",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 200000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 4000, Disk: 100000, GPU: 0},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 8, Memory: 32000, Disk: 100000, GPU: 2},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 50000, GPU: 1},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 2, Memory: 16000, Disk: 150000, GPU: 4},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 4000, Disk: 20000, GPU: 2},
				},
			},
			jobResources: models.ResourcesConfig{
				Disk: "10mb",
			},
			expected: []string{"node1", "node3", "node2"},
		},
		{
			name: "High GPU weight",
			nodes: []nodeScenario{
				{
					nodeID:            "node1",
					availableCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 100000, GPU: 4},
					queueUsedCapacity: models.Resources{CPU: 1, Memory: 4000, Disk: 50000, GPU: 1},
				},
				{
					nodeID:            "node2",
					availableCapacity: models.Resources{CPU: 8, Memory: 32000, Disk: 150000, GPU: 2},
					queueUsedCapacity: models.Resources{CPU: 2, Memory: 8000, Disk: 30000, GPU: 0},
				},
				{
					nodeID:            "node3",
					availableCapacity: models.Resources{CPU: 4, Memory: 16000, Disk: 200000, GPU: 1},
					queueUsedCapacity: models.Resources{CPU: 0, Memory: 2000, Disk: 10000, GPU: 0},
				},
			},
			jobResources: models.ResourcesConfig{
				GPU: "1",
			},
			expected: []string{"node1", "node2", "node3"},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			nodes := make([]models.NodeInfo, len(tc.nodes))
			for i, ns := range tc.nodes {
				nodes[i] = models.NodeInfo{
					NodeID: ns.nodeID,
					ComputeNodeInfo: &models.ComputeNodeInfo{
						AvailableCapacity: ns.availableCapacity,
						QueueUsedCapacity: ns.queueUsedCapacity,
					},
				}
			}

			job := mock.Job()
			job.Task().ResourcesConfig = &tc.jobResources
			ranks, err := suite.ranker.RankNodes(context.Background(), *job, nodes)
			suite.NoError(err)

			// sort nodes by rank
			sort.Slice(ranks, func(i, j int) bool {
				return ranks[i].Rank > ranks[j].Rank
			})

			prevRank := -1
			for _, rank := range ranks {
				suite.Require().GreaterOrEqual(rank.Rank, 0)
				suite.Require().LessOrEqual(rank.Rank, maxAvailableCapacityRank+maxQueueCapacityRank)
				if prevRank == -1 {
					prevRank = rank.Rank
				} else {
					if tc.equalCheck {
						suite.Equal(prevRank, rank.Rank)
					} else {
						suite.Greater(prevRank, rank.Rank)
						prevRank = rank.Rank
					}
				}
			}
		})
	}
}

func TestAvailableCapacityNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(AvailableCapacityNodeRankerSuite))
}
