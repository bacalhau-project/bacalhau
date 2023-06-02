package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	engtypes "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/types"
	strgtypes "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/types"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
)

type NodeInfoProviderParams struct {
	Executors          executor.ExecutorProvider
	Verifiers          verifier.VerifierProvider
	Publisher          publisher.PublisherProvider
	Storages           storage.StorageProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements model.ResourceUsageData
}

type NodeInfoProvider struct {
	executors          executor.ExecutorProvider
	verifiers          verifier.VerifierProvider
	publishers         publisher.PublisherProvider
	storages           storage.StorageProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements model.ResourceUsageData
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		executors:          params.Executors,
		verifiers:          params.Verifiers,
		publishers:         params.Publisher,
		storages:           params.Storages,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (n *NodeInfoProvider) GetComputeInfo(ctx context.Context) model.ComputeNodeInfo {
	return model.ComputeNodeInfo{
		ExecutionEngines:   model.InstalledTypes(ctx, n.executors, engtypes.EngineTypes()),
		Verifiers:          model.InstalledTypes(ctx, n.verifiers, model.VerifierTypes()),
		Publishers:         model.InstalledTypes(ctx, n.publishers, model.PublisherTypes()),
		StorageSources:     model.InstalledTypes(ctx, n.storages, strgtypes.StorageTypes()),
		MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
	}
}

// compile-time interface check
var _ model.ComputeNodeInfoProvider = &NodeInfoProvider{}
