package compute

import (
	"context"
	"sync"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

type NodeInfoPublisherParams struct {
	PubSub             pubsub.PubSub[model.NodeInfo]
	Host               host.Host
	Executors          executor.ExecutorProvider
	CapacityTracker    capacity.Tracker
	ExecutorBuffer     *ExecutorBuffer
	MaxJobRequirements model.ResourceUsageData
	Interval           time.Duration
}

type NodeInfoPublisher struct {
	pubSub             pubsub.PubSub[model.NodeInfo]
	h                  host.Host
	executors          executor.ExecutorProvider
	capacityTracker    capacity.Tracker
	executorBuffer     *ExecutorBuffer
	maxJobRequirements model.ResourceUsageData
	interval           time.Duration

	stopChannel chan struct{}
	stopOnce    sync.Once
}

func NewNodeInfoPublisher(params NodeInfoPublisherParams) *NodeInfoPublisher {
	p := &NodeInfoPublisher{
		pubSub:             params.PubSub,
		h:                  params.Host,
		executors:          params.Executors,
		capacityTracker:    params.CapacityTracker,
		executorBuffer:     params.ExecutorBuffer,
		maxJobRequirements: params.MaxJobRequirements,
		interval:           params.Interval,
		stopChannel:        make(chan struct{}),
	}

	go p.publishBackgroundTask()
	return p
}

// Publish publishes the node info to the pubsub topic manually and won't wait for the background task to do it.
func (n *NodeInfoPublisher) Publish(ctx context.Context) error {
	var executionEngines []model.Engine
	for _, e := range model.EngineTypes() {
		if n.executors.HasExecutor(ctx, e) {
			executionEngines = append(executionEngines, e)
		}
	}

	nodeInfo := model.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID:    n.h.ID(),
			Addrs: n.h.Addrs(),
		},
		NodeType: model.NodeTypeCompute,
		ComputeNodeInfo: model.ComputeNodeInfo{
			ExecutionEngines:   executionEngines,
			MaxCapacity:        n.capacityTracker.GetMaxCapacity(ctx),
			AvailableCapacity:  n.capacityTracker.GetAvailableCapacity(ctx),
			MaxJobRequirements: n.maxJobRequirements,
			RunningExecutions:  len(n.executorBuffer.RunningExecutions()),
			EnqueuedExecutions: len(n.executorBuffer.EnqueuedExecutions()),
		},
	}

	return n.pubSub.Publish(ctx, nodeInfo)
}

func (n *NodeInfoPublisher) publishBackgroundTask() {
	ctx := context.Background()
	ticker := time.NewTicker(n.interval)
	for {
		select {
		case <-ticker.C:
			err := n.Publish(ctx)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to publish node info")
			}
		case <-n.stopChannel:
			log.Ctx(ctx).Info().Msg("stopped publishing node info")
			ticker.Stop()
			return
		}
	}
}

// Stop stops the background task that publishes the node info periodically
func (n *NodeInfoPublisher) Stop() {
	n.stopOnce.Do(func() {
		n.stopChannel <- struct{}{}
	})
	log.Info().Msg("done stopping compute node info publisher")
}
