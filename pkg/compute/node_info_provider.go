package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type NodeInfoProviderParams struct {
	Executors          executor.ExecutorProvider
	Publisher          publisher.PublisherProvider
	Storages           storage.StorageProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements models.Resources
}

type NodeInfoProvider struct {
	executors          executor.ExecutorProvider
	publishers         publisher.PublisherProvider
	storages           storage.StorageProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements models.Resources
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		executors:          params.Executors,
		publishers:         params.Publisher,
		storages:           params.Storages,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (n *NodeInfoProvider) GetComputeInfo(ctx context.Context) models.ComputeNodeInfo {
	return models.ComputeNodeInfo{
		ExecutionEngines:   models.InstalledTypes(ctx, n.executors, models.EngineTypes()),
		Publishers:         models.InstalledTypes(ctx, n.publishers, models.PublisherTypes()),
		StorageSources:     models.InstalledTypes(ctx, n.storages, models.StorageSourceTypes()),
		MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
	}
}

// compile-time interface check
var _ models.ComputeNodeInfoProvider = &NodeInfoProvider{}
