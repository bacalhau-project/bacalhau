package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type NodeInfoDecoratorParams struct {
	Executors          executor.ExecutorProvider
	Publisher          publisher.PublisherProvider
	Storages           storage.StorageProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements models.Resources
}

type NodeInfoDecorator struct {
	executors          executor.ExecutorProvider
	publishers         publisher.PublisherProvider
	storages           storage.StorageProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements models.Resources
}

func NewNodeInfoDecorator(params NodeInfoDecoratorParams) *NodeInfoDecorator {
	return &NodeInfoDecorator{
		executors:          params.Executors,
		publishers:         params.Publisher,
		storages:           params.Storages,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (n *NodeInfoDecorator) DecorateNodeInfo(ctx context.Context, nodeInfo models.NodeInfo) models.NodeInfo {
	nodeInfo.NodeType = models.NodeTypeCompute
	nodeInfo.ComputeNodeInfo = &models.ComputeNodeInfo{
		ExecutionEngines:   n.executors.Keys(ctx),
		Publishers:         n.publishers.Keys(ctx),
		StorageSources:     n.storages.Keys(ctx),
		MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: n.executorBuffer.EnqueuedExecutionsCount(),
	}
	return nodeInfo
}

// compile-time interface check
var _ models.NodeInfoDecorator = &NodeInfoDecorator{}
