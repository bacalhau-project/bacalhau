package analytics

import (
	"context"
	"fmt"
	"strings"
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

	// Utilization percentages (based on used/total capacity)
	s.InDelta(41.67, props["utilization_cpu_percent"], 0.01)
	s.InDelta(25.0, props["utilization_memory_percent"], 0.1)
	s.InDelta(25.0, props["utilization_disk_percent"], 0.1)
	s.InDelta(50.0, props["utilization_gpu_percent"], 0.1)
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

func (s *NodeTestSuite) TestCapabilitiesSortingByCount() {
	// Create nodes with different execution engines to test sorting
	node1 := s.createTestNodeState()
	node2 := s.createTestNodeState()
	node3 := s.createTestNodeState()
	node4 := s.createTestNodeState()

	// Set up different execution engines with varying counts
	node1.Info.ComputeNodeInfo.ExecutionEngines = []string{"docker", "wasm"}   // docker: 4, wasm: 2
	node2.Info.ComputeNodeInfo.ExecutionEngines = []string{"docker", "podman"} // podman: 1
	node3.Info.ComputeNodeInfo.ExecutionEngines = []string{"docker", "wasm"}
	node4.Info.ComputeNodeInfo.ExecutionEngines = []string{"docker"}

	nodes := []models.NodeState{node1, node2, node3, node4}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// Verify capabilities are sorted by count (docker=4, wasm=2, podman=1)
	s.Equal(4, props["capabilities_execution_engines_docker"])
	s.Equal(2, props["capabilities_execution_engines_wasm"])
	s.Equal(1, props["capabilities_execution_engines_podman"])
	s.Equal(3, props["capabilities_execution_engines_count"])
}

func (s *NodeTestSuite) TestLargeResourceValuesNoOverflow() {
	// Test with very large resource values that could cause overflow
	node1 := s.createTestNodeState()
	node2 := s.createTestNodeState()

	// Set very large resource values (close to uint64 max for memory/disk)
	largeMemory := uint64(1 << 60) // 1 exabyte
	largeDisk := uint64(1 << 61)   // 2 exabytes

	node1.Info.ComputeNodeInfo.MaxCapacity.Memory = largeMemory
	node1.Info.ComputeNodeInfo.MaxCapacity.Disk = largeDisk
	node1.Info.ComputeNodeInfo.AvailableCapacity.Memory = largeMemory / 2
	node1.Info.ComputeNodeInfo.AvailableCapacity.Disk = largeDisk / 2

	node2.Info.ComputeNodeInfo.MaxCapacity.Memory = largeMemory
	node2.Info.ComputeNodeInfo.MaxCapacity.Disk = largeDisk
	node2.Info.ComputeNodeInfo.AvailableCapacity.Memory = largeMemory / 3
	node2.Info.ComputeNodeInfo.AvailableCapacity.Disk = largeDisk / 3

	nodes := []models.NodeState{node1, node2}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// Should not panic and should produce reasonable values
	s.Equal(largeMemory*2, props["resources_total_memory"])
	s.Equal(largeDisk*2, props["resources_total_disk"])

	// Standard deviation should be calculated without overflow
	s.NotNil(props["resource_stats_std_dev_memory"])
	s.NotNil(props["resource_stats_std_dev_disk"])

	// Values should be finite (not NaN or Inf)
	memStdDev, ok := props["resource_stats_std_dev_memory"].(uint64)
	s.True(ok)
	s.True(memStdDev < uint64(largeMemory)) // Reasonable value
}

func (s *NodeTestSuite) TestNoopEventFiltering() {
	// Test that NoopEvent has the correct type
	s.Equal(NoopEventType, NoopEvent.Type())
	s.NotNil(NoopEvent.Properties())
	s.Equal(EventProperties{}, NoopEvent.Properties())
}

func (s *NodeTestSuite) TestMixedComputeAndRequesterNodes() {
	// Test filtering works correctly with mixed node types
	computeNode := s.createTestNodeState()
	requesterNode := s.createTestNodeState()
	requesterNode.Info.NodeType = models.NodeTypeRequester

	// Mix of compute and requester nodes
	nodes := []models.NodeState{computeNode, requesterNode}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// Should only count the compute node
	s.Equal(1, props["total_nodes"])
	s.Equal(2.0, props["resources_total_cpu"]) // Only from compute node
}

func (s *NodeTestSuite) TestUtilizationCalculationEdgeCases() {
	// Test edge case where available > max capacity (shouldn't happen but handle gracefully)
	node := s.createTestNodeState()

	// Set available higher than max (edge case)
	node.Info.ComputeNodeInfo.AvailableCapacity.CPU = 5.0
	node.Info.ComputeNodeInfo.MaxCapacity.CPU = 2.0

	nodes := []models.NodeState{node}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// The existing Sub method should handle this by setting used capacity to 0
	// when available > max, resulting in 0% utilization
	utilization, ok := props["utilization_cpu_percent"].(float64)
	s.True(ok)
	s.Equal(0.0, utilization) // Should be 0% when available > max
}

func (s *NodeTestSuite) TestMaxCapabilitiesLimit() {
	// Test that we don't exceed maxEnginesToReport (10) capabilities
	node := s.createTestNodeState()

	// Create more than 10 execution engines
	engines := make([]string, 15)
	for i := 0; i < 15; i++ {
		engines[i] = fmt.Sprintf("engine%d", i)
	}
	node.Info.ComputeNodeInfo.ExecutionEngines = engines

	nodes := []models.NodeState{node}
	event := NewNodeInfosEvent(nodes)

	props := event.Properties()

	// Should report exactly 15 distinct engines
	s.Equal(15, props["capabilities_execution_engines_count"])

	// But should only include details for max 10
	engineCount := 0
	for key := range props {
		if strings.HasPrefix(key, "capabilities_execution_engines_") &&
			key != "capabilities_execution_engines_count" {
			engineCount++
		}
	}
	s.Equal(10, engineCount) // Should be limited to maxEnginesToReport
}
