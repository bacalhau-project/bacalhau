package compute

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type NodeInfoProviderParams struct {
	Executors          executor.ExecutorProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements model.ResourceUsageData
}

type NodeInfoProvider struct {
	executors          executor.ExecutorProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements model.ResourceUsageData
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		executors:          params.Executors,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (n *NodeInfoProvider) GetComputeInfo(ctx context.Context) model.ComputeNodeInfo {
	var executionEngines []model.Engine
	for _, e := range model.EngineTypes() {
		if n.executors.Has(ctx, e) {
			executionEngines = append(executionEngines, e)
		}
	}

	return model.ComputeNodeInfo{
		ExecutionEngines:   executionEngines,
		MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
	}
}

// compile-time interface check
var _ model.ComputeNodeInfoProvider = &NodeInfoProvider{}
