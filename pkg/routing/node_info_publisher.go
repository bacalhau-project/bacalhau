package routing

import (
	"context"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type NodeInfoPublisherParams struct {
	PubSub           pubsub.PubSub[model.NodeInfo]
	NodeInfoProvider model.NodeInfoProvider
	Interval         time.Duration
}

type NodeInfoPublisher struct {
	pubSub           pubsub.PubSub[model.NodeInfo]
	nodeInfoProvider model.NodeInfoProvider
	interval         time.Duration

	stopChannel chan struct{}
	stopOnce    sync.Once
}

func NewNodeInfoPublisher(params NodeInfoPublisherParams) *NodeInfoPublisher {
	p := &NodeInfoPublisher{
		pubSub:           params.PubSub,
		nodeInfoProvider: params.NodeInfoProvider,
		interval:         params.Interval,
		stopChannel:      make(chan struct{}),
	}

	go p.publishBackgroundTask()
	return p
}

// Publish publishes the node info to the pubsub topic manually and won't wait for the background task to do it.
func (n *NodeInfoPublisher) Publish(ctx context.Context) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/routing.NodeInfoPublisher.Publish")
	defer span.End()

	return n.pubSub.Publish(ctx, n.nodeInfoProvider.GetNodeInfo(ctx))
}

func (n *NodeInfoPublisher) publishBackgroundTask() {
	ctx := context.Background()
	ticker := time.NewTicker(n.interval)
	for {
		select {
		case <-ticker.C:
			func() {
				ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/routing.NodeInfoPublisher.publishBackgroundTask") //nolint:govet
				defer span.End()

				err := n.Publish(ctx)
				if err != nil {
					log.Ctx(ctx).Err(err).Msg("failed to publish node info")
				}
			}()
		case <-n.stopChannel:
			log.Ctx(ctx).Debug().Msg("stopped publishing node info")
			ticker.Stop()
			return
		}
	}
}

// Stop stops the background task that publishes the node info periodically
func (n *NodeInfoPublisher) Stop(ctx context.Context) {
	n.stopOnce.Do(func() {
		n.stopChannel <- struct{}{}
	})
	log.Ctx(ctx).Debug().Msg("done stopping compute node info publisher")
}
