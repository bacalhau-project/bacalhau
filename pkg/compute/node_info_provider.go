package compute

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
)

type NodeInfoProviderParams struct {
	Host               host.Host
	IdentityService    identify.IDService
	Labels             map[string]string
	Executors          executor.ExecutorProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements model.ResourceUsageData
}

type NodeInfoProvider struct {
	h                  host.Host
	identityService    identify.IDService
	labels             map[string]string
	executors          executor.ExecutorProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements model.ResourceUsageData
}

func NewNodeInfoProvider(params NodeInfoProviderParams) *NodeInfoProvider {
	return &NodeInfoProvider{
		h:                  params.Host,
		identityService:    params.IdentityService,
		labels:             params.Labels,
		executors:          params.Executors,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
	}
}

func (n *NodeInfoProvider) GetNodeInfo(ctx context.Context) model.NodeInfo {
	var executionEngines []model.Engine
	for _, e := range model.EngineTypes() {
		if n.executors.HasExecutor(ctx, e) {
			executionEngines = append(executionEngines, e)
		}
	}

	return model.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.identityService.OwnObservedAddrs(),
		},
		NodeType: model.NodeTypeCompute,
		Labels:   n.labels,
		ComputeNodeInfo: model.ComputeNodeInfo{
			ExecutionEngines:   executionEngines,
			MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
			AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
			MaxJobRequirements: n.maxJobRequirements,
			RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
			EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
		},
	}
}

// compile-time interface check
var _ model.NodeInfoProvider = &NodeInfoProvider{}
