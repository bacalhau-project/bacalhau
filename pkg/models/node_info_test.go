//go:build unit || !integration

package models

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type NodeInfoTestSuite struct {
	suite.Suite
}

func TestNodeInfoTestSuite(t *testing.T) {
	suite.Run(t, new(NodeInfoTestSuite))
}

func (s *NodeInfoTestSuite) TestHasNodeInfoChanged() {
	baseNodeInfo := &NodeInfo{
		NodeID:   "node-1",
		NodeType: NodeTypeCompute,
		Labels: map[string]string{
			"zone": "us-east-1",
			"env":  "prod",
		},
		SupportedProtocols: []Protocol{ProtocolNCLV1, ProtocolBProtocolV2},
		BacalhauVersion:    BuildVersionInfo{Major: "1", Minor: "0"},
		ComputeNodeInfo: ComputeNodeInfo{
			ExecutionEngines: []string{"docker", "wasm"},
			Publishers:       []string{"ipfs"},
			StorageSources:   []string{"s3", "ipfs"},
			MaxCapacity: Resources{
				CPU:    4,
				Memory: 8192,
				Disk:   100,
				GPU:    1,
				GPUs: []GPU{
					{
						Index:      0,
						Name:       "Tesla T4",
						Vendor:     GPUVendorNvidia,
						Memory:     16384,
						PCIAddress: "0000:00:1e.0",
					},
				},
			},
			MaxJobRequirements: Resources{
				CPU:    2,
				Memory: 4096,
				Disk:   50,
				GPU:    0,
			},
			// Dynamic fields that should be ignored
			QueueUsedCapacity: Resources{
				CPU:    1,
				Memory: 2048,
				Disk:   25,
			},
			AvailableCapacity: Resources{
				CPU:    3,
				Memory: 6144,
				Disk:   75,
			},
			RunningExecutions:  2,
			EnqueuedExecutions: 1,
		},
	}

	testCases := []struct {
		name           string
		changeFunction func(info *NodeInfo) *NodeInfo
		expectChanged  bool
	}{
		{
			name:           "identical nodes",
			changeFunction: func(info *NodeInfo) *NodeInfo { return info.Copy() },
			expectChanged:  false,
		},
		{
			name: "different node ID",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.NodeID = "node-2"
				return info
			},
			expectChanged: true,
		},
		{
			name: "different node type",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.NodeType = NodeTypeRequester
				return info
			},
			expectChanged: true,
		},
		{
			name: "different labels - new label",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.Labels["new"] = "value"
				return info
			},
			expectChanged: true,
		},
		{
			name: "different labels - changed value",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.Labels["zone"] = "us-west-1"
				return info
			},
			expectChanged: true,
		},
		{
			name: "different labels - removed label",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				delete(info.Labels, "zone")
				return info
			},
			expectChanged: true,
		},
		{
			name: "different protocols - added",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.SupportedProtocols = append(info.SupportedProtocols, Protocol("NewProtocol"))
				return info
			},
			expectChanged: true,
		},
		{
			name: "different version",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.BacalhauVersion.Minor = "1"
				return info
			},
			expectChanged: true,
		},
		{
			name: "different execution engines",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.ExecutionEngines = append(info.ComputeNodeInfo.ExecutionEngines, "kubernetes")
				return info
			},
			expectChanged: true,
		},
		{
			name: "different publishers",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.Publishers = append(info.ComputeNodeInfo.Publishers, "s3")
				return info
			},
			expectChanged: true,
		},
		{
			name: "different storage sources",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.StorageSources = []string{"s3"}
				return info
			},
			expectChanged: true,
		},
		{
			name: "different max capacity",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.MaxCapacity.CPU = 8
				return info
			},
			expectChanged: true,
		},
		{
			name: "different max job requirements",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.MaxJobRequirements.Memory = 8192
				return info
			},
			expectChanged: true,
		},
		{
			name: "changed queue capacity only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.QueueUsedCapacity.CPU = 2
				return info
			},
			expectChanged: false,
		},
		{
			name: "changed available capacity only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.AvailableCapacity.Memory = 4096
				return info
			},
			expectChanged: false,
		},
		{
			name: "changed running executions only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.RunningExecutions = 5
				return info
			},
			expectChanged: false,
		},
		{
			name: "changed enqueued executions only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.EnqueuedExecutions = 3
				return info
			},
			expectChanged: false,
		},
		{
			name: "multiple dynamic field changes only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.RunningExecutions = 5
				info.ComputeNodeInfo.EnqueuedExecutions = 3
				info.ComputeNodeInfo.QueueUsedCapacity.CPU = 2
				info.ComputeNodeInfo.AvailableCapacity.Memory = 4096
				return info
			},
			expectChanged: false,
		},
		{
			name: "same labels different order",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				// Recreate labels in different order
				info.Labels = map[string]string{
					"env":  "prod",
					"zone": "us-east-1",
				}
				return info
			},
			expectChanged: false,
		},
		{
			name: "same engines different order",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.ExecutionEngines = []string{"wasm", "docker"}
				return info
			},
			expectChanged: false,
		},
		{
			name: "same storage sources different order",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.StorageSources = []string{"ipfs", "s3"}
				return info
			},
			expectChanged: false,
		},
		{
			name: "same protocols different order",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.SupportedProtocols = []Protocol{ProtocolBProtocolV2, ProtocolNCLV1}
				return info
			},
			expectChanged: false,
		},
		{
			name: "different max capacity GPUs",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.MaxCapacity.GPUs = []GPU{
					{Index: 0, Name: "RTX 3080", Vendor: GPUVendorNvidia, Memory: 10240, PCIAddress: "0000:00:1e.0"}, // Different GPU spec
				}
				return info
			},
			expectChanged: true,
		},
		{
			name: "same GPUs different order",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				// Set the exact same GPUs but in different order
				info.ComputeNodeInfo.MaxCapacity.GPUs = []GPU{
					{Index: 0, Name: "Tesla T4", Vendor: GPUVendorNvidia, Memory: 16384, PCIAddress: "0000:00:1e.0"},
				}
				return info
			},
			expectChanged: false,
		},
		{
			name: "changed GPU count only",
			changeFunction: func(info *NodeInfo) *NodeInfo {
				info = info.Copy()
				info.ComputeNodeInfo.MaxCapacity.GPU = 2
				return info
			},
			expectChanged: true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			current := tc.changeFunction(baseNodeInfo)
			changed := baseNodeInfo.HasStaticConfigChanged(*current)

			if tc.expectChanged {
				s.True(changed, "Expected node info to have changed")
			} else {
				s.False(changed, "Expected node info to remain unchanged")
			}
		})
	}
}
