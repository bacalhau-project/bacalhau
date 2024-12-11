package test

import (
	"context"
	"sync"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type MockNodeInfoProvider struct {
	nodeInfo models.NodeInfo
	mu       sync.RWMutex
}

// NewMockNodeInfoProvider creates a new mock node info provider
func NewMockNodeInfoProvider() *MockNodeInfoProvider {
	return &MockNodeInfoProvider{
		nodeInfo: models.NodeInfo{
			NodeID:   "test-node",
			NodeType: models.NodeTypeCompute,
			Labels:   map[string]string{},
			ComputeNodeInfo: models.ComputeNodeInfo{
				AvailableCapacity: models.Resources{CPU: 4},
				QueueUsedCapacity: models.Resources{CPU: 1},
			},
		},
	}
}

func (m *MockNodeInfoProvider) GetNodeInfo(ctx context.Context) models.NodeInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.nodeInfo
}

func (m *MockNodeInfoProvider) SetNodeInfo(nodeInfo models.NodeInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeInfo = nodeInfo
}
