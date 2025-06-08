package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type NodeTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestNodeSuite(t *testing.T) {
	suite.Run(t, new(NodeTestSuite))
}

func (s *NodeTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *NodeTestSuite) createTestNodeState() models.NodeState {
	return models.NodeState{
		Info: models.NodeInfo{
			NodeID:   "test-node-id",
			NodeType: models.NodeTypeCompute,
			Labels:   map[string]string{"region": "test-region"},
			BacalhauVersion: models.BuildVersionInfo{
				GitVersion: "v1.0.0",
				GitCommit:  "abc123",
				Major:      "1",
				Minor:      "0",
				BuildDate:  time.Now(),
				GOOS:       "linux",
				GOARCH:     "amd64",
			},
			ComputeNodeInfo: models.ComputeNodeInfo{
				MaxCapacity: models.Resources{
					CPU:    2.0,
					Memory: 4096,
					Disk:   8192,
					GPU:    2,
					GPUs: []models.GPU{
						{Name: "test-gpu", Vendor: models.GPUVendorNvidia},
					},
				},
				AvailableCapacity: models.Resources{
					CPU:    1.5,
					Memory: 3072,
					Disk:   6144,
					GPU:    1,
					GPUs: []models.GPU{
						{Name: "test-gpu", Vendor: models.GPUVendorNvidia},
					},
				},
				ExecutionEngines:   []string{"docker", "wasm"},
				StorageSources:     []string{"ipfs", "s3"},
				Publishers:         []string{"ipfs", "s3"},
				RunningExecutions:  2,
				EnqueuedExecutions: 1,
			},
			SupportedProtocols: []models.Protocol{models.ProtocolBProtocolV2, models.ProtocolNCLV1},
		},
		ConnectionState: models.ConnectionState{
			Status: models.NodeStates.CONNECTED,
		},
	}
}

func (s *NodeTestSuite) TestNodeInfosEvent() {
	// Create multiple nodes to test aggregation logic
	node1 := s.createTestNodeState()
	node2 := s.createTestNodeState()

	// Modify second node to have different values for testing variety
	node2.Info.ComputeNodeInfo.MaxCapacity.CPU = 4.0
	node2.Info.ComputeNodeInfo.AvailableCapacity.CPU = 2.0
	node2.Info.ComputeNodeInfo.RunningExecutions = 3
	node2.ConnectionState.Status = models.NodeStates.DISCONNECTED

	// Create event with both nodes
	nodes := []models.NodeState{node1, node2}
	event := NewNodeInfosEvent(nodes)

	s.Equal(NodeInfoEventType, event.Type())

	props := event.Properties()

	// Test basic node metrics
	s.Equal(2, props["total_nodes"])

	// Test node state properties
	s.Equal(1, props["nodes_by_state_connected"])
	s.Equal(1, props["nodes_by_state_disconnected"])

	// Test resource properties
	s.Equal(6.0, props["resources_total_cpu"])
	s.Equal(uint64(8192), props["resources_total_memory"])
	s.Equal(uint64(16384), props["resources_total_disk"])
	s.Equal(uint64(4), props["resources_total_gpu"])

	s.Equal(3.5, props["resources_available_cpu"])
	s.Equal(uint64(6144), props["resources_available_memory"])
	s.Equal(uint64(12288), props["resources_available_disk"])
	s.Equal(uint64(2), props["resources_available_gpu"])

	// Test resource stats properties
	s.Equal(1.5, props["resource_stats_min_cpu"])
	s.Equal(uint64(3072), props["resource_stats_min_memory"])
	s.Equal(uint64(6144), props["resource_stats_min_disk"])
	s.Equal(uint64(1), props["resource_stats_min_gpu"])

	s.Equal(2.0, props["resource_stats_max_cpu"])
	s.Equal(uint64(3072), props["resource_stats_max_memory"])
	s.Equal(uint64(6144), props["resource_stats_max_disk"])
	s.Equal(uint64(1), props["resource_stats_max_gpu"])

	s.Equal(1.75, props["resource_stats_avg_cpu"])
	s.Equal(uint64(3072), props["resource_stats_avg_memory"])
	s.Equal(uint64(6144), props["resource_stats_avg_disk"])
	s.Equal(uint64(1), props["resource_stats_avg_gpu"])

	// Standard deviation should be calculated correctly
	s.InDelta(0.35355, props["resource_stats_std_dev_cpu"], 0.00001)
	s.Equal(uint64(0), props["resource_stats_std_dev_memory"])
	s.Equal(uint64(0), props["resource_stats_std_dev_disk"])
	s.Equal(uint64(0), props["resource_stats_std_dev_gpu"])

	// Test capabilities properties
	s.Equal(2, props["capabilities_execution_engines_count"])
	s.Equal(2, props["capabilities_execution_engines_docker"])
	s.Equal(2, props["capabilities_execution_engines_wasm"])

	s.Equal(2, props["capabilities_storage_sources_count"])
	s.Equal(2, props["capabilities_storage_sources_ipfs"])
	s.Equal(2, props["capabilities_storage_sources_s3"])

	s.Equal(2, props["capabilities_publishers_count"])
	s.Equal(2, props["capabilities_publishers_ipfs"])
	s.Equal(2, props["capabilities_publishers_s3"])

	s.Equal(2, props["capabilities_protocols_count"])
	s.Equal(2, props["capabilities_protocols_bprotocol/v2"])
	s.Equal(2, props["capabilities_protocols_ncl/v1"])

	s.Equal(1, props["capabilities_versions_count"])
	s.Equal(2, props["capabilities_versions_v1.0.0"])

	// Test utilization properties
	s.Equal(5, props["utilization_total_running_executions"])
	s.Equal(2, props["utilization_total_enqueued_executions"])
	s.Equal(2.5, props["utilization_avg_running_executions"])
	s.Equal(1.0, props["utilization_avg_enqueued_executions"])

	// Utilization percentages
	s.InDelta(83.33, props["utilization_cpu_percent"], 0.01)
	s.InDelta(0.061, props["utilization_memory_percent"], 0.001)
	s.InDelta(0.031, props["utilization_disk_percent"], 0.001)
	s.InDelta(125.0, props["utilization_gpu_percent"], 0.1)
}

func (s *NodeTestSuite) TestEmptyNodeInfosEvent() {
	// Test with empty node list
	emptyEvent := NewNodeInfosEvent([]models.NodeState{})
	s.Equal(NoopEventType, emptyEvent.Type())
}

func (s *NodeTestSuite) TestNonComputeNodeInfosEvent() {
	// Create a non-compute node
	nonComputeNode := s.createTestNodeState()
	nonComputeNode.Info.NodeType = models.NodeTypeRequester

	// Event should filter out non-compute nodes
	event := NewNodeInfosEvent([]models.NodeState{nonComputeNode})
	s.Equal(NoopEventType, event.Type())
}

func (s *NodeTestSuite) TestZeroResourceNodeInfosEvent() {
	// Create a node with zero resources
	zeroNode := s.createTestNodeState()
	zeroNode.Info.ComputeNodeInfo.MaxCapacity = models.Resources{}
	zeroNode.Info.ComputeNodeInfo.AvailableCapacity = models.Resources{}

	// The event should handle zero values properly
	event := NewNodeInfosEvent([]models.NodeState{zeroNode})

	s.Equal(NodeInfoEventType, event.Type())

	props := event.Properties()

	// Resource totals should be zero
	s.Equal(0.0, props["resources_total_cpu"])
	s.Equal(uint64(0), props["resources_total_memory"])
	s.Equal(uint64(0), props["resources_total_disk"])
	s.Equal(uint64(0), props["resources_total_gpu"])

	// Resource stats should handle zero values
	s.Equal(0.0, props["resource_stats_min_cpu"])
	s.Equal(0.0, props["resource_stats_max_cpu"])
	s.Equal(0.0, props["resource_stats_avg_cpu"])
	s.Equal(0.0, props["resource_stats_std_dev_cpu"])

	// Utilization percentages should be zero to avoid division by zero
	s.Equal(0.0, props["utilization_cpu_percent"])
	s.Equal(0.0, props["utilization_memory_percent"])
	s.Equal(0.0, props["utilization_disk_percent"])
	s.Equal(0.0, props["utilization_gpu_percent"])
}

func (s *NodeTestSuite) TestMultiNodeStdDevCalculation() {
	// Create nodes with varied CPU values to test std dev calculation
	node1 := s.createTestNodeState()
	node2 := s.createTestNodeState()
	node3 := s.createTestNodeState()

	node1.Info.ComputeNodeInfo.AvailableCapacity.CPU = 2.0
	node2.Info.ComputeNodeInfo.AvailableCapacity.CPU = 4.0
	node3.Info.ComputeNodeInfo.AvailableCapacity.CPU = 6.0

	nodes := []models.NodeState{node1, node2, node3}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// Mean should be 4.0
	s.Equal(4.0, props["resource_stats_avg_cpu"])

	// Standard deviation should be 2.0
	s.InDelta(2.0, props["resource_stats_std_dev_cpu"], 0.00001)
}
