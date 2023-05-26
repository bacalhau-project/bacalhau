package compute

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/git"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/gitlfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
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
		ExecutionEngines: model.InstalledTypes(ctx, n.executors, []cid.Cid{
			wasm.EngineSchema.Cid(),
			docker.EngineSchema.Cid(),
		}),
		Verifiers:  model.InstalledTypes(ctx, n.verifiers, model.VerifierTypes()),
		Publishers: model.InstalledTypes(ctx, n.publishers, model.PublisherTypes()),
		StorageSources: model.InstalledTypes(ctx, n.storages, []cid.Cid{
			s3.Schema.Cid(),
			url.Schema.Cid(),
			git.Schema.Cid(),
			ipfs.Schema.Cid(),
			local.Schema.Cid(),
			gitlfs.Schema.Cid(),
			inline.Schema.Cid(),
		}),
		MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
		AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
		MaxJobRequirements: n.maxJobRequirements,
		RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
		EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
	}
}

// compile-time interface check
var _ model.ComputeNodeInfoProvider = &NodeInfoProvider{}
