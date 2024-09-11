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
	Executors              executor.ExecutorProvider
	Publisher              publisher.PublisherProvider
	Storages               storage.StorageProvider
	RunningCapacityTracker capacity.Tracker
	QueueCapacityTracker   capacity.UsageTracker
	ExecutorBuffer         *ExecutorBuffer
	MaxJobRequirements     models.Resources
}

type NodeInfoDecorator struct {
	executors              executor.ExecutorProvider
	publishers             publisher.PublisherProvider
	storages               storage.StorageProvider
	runningCapacityTracker capacity.Tracker
	queueCapacityTracker   capacity.UsageTracker
	executorBuffer         *ExecutorBuffer
	maxJobRequirements     models.Resources
}

func NewNodeInfoDecorator(params NodeInfoDecoratorParams) *NodeInfoDecorator {
	return &NodeInfoDecorator{
		executors:              params.Executors,
		publishers:             params.Publisher,
		storages:               params.Storages,
		runningCapacityTracker: params.RunningCapacityTracker,
		queueCapacityTracker:   params.QueueCapacityTracker,
		executorBuffer:         params.ExecutorBuffer,
		maxJobRequirements:     params.MaxJobRequirements,
	}
}

func (n *NodeInfoDecorator) DecorateNodeInfo(ctx context.Context, nodeInfo models.NodeInfo) models.NodeInfo {
	// TODO(forrest): this method takes 10 seconds to run: https://github.com/bacalhau-project/bacalhau/issues/4153
	// because the Keys() methods are slow when s3 is considered since we need to check for credentials.
	nodeInfo.NodeType = models.NodeTypeCompute
	nodeInfo.ComputeNodeInfo = &models.ComputeNodeInfo{
		ExecutionEngines:   n.executors.Keys(ctx),
		Publishers:         n.publishers.Keys(ctx),
		StorageSources:     n.storages.Keys(ctx),
		MaxCapacity:        n.runningCapacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.runningCapacityTracker.GetAvailableCapacity(ctx),
		QueueUsedCapacity:  n.queueCapacityTracker.GetUsedCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: n.executorBuffer.EnqueuedExecutionsCount(),
	}
	return nodeInfo
}

// compile-time interface check
var _ models.NodeInfoDecorator = &NodeInfoDecorator{}
