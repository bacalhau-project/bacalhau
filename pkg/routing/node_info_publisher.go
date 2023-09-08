package routing

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/pubsub"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type NodeInfoPublisherIntervalConfig struct {
	// Interval is the interval between publishing node info
	Interval time.Duration

	// During node startup, we can publish node info more frequently to speed up the discovery process.
	// EagerPublishInterval is the interval between publishing node info during startup.
	EagerPublishInterval time.Duration
	// EagerPublishDuration is the duration of the eager publish period. After this period, the node will publish node info
	// with the standard interval.
	EagerPublishDuration time.Duration
}

// IsZero returns true if the interval config is zero
func (n NodeInfoPublisherIntervalConfig) IsZero() bool {
	return n.Interval == 0 && n.EagerPublishInterval == 0 && n.EagerPublishDuration == 0
}

// IsEagerPublishEnabled returns true if eager publish is enabled
func (n NodeInfoPublisherIntervalConfig) IsEagerPublishEnabled() bool {
	return n.EagerPublishInterval > 0 && n.EagerPublishDuration > 0
}

type NodeInfoPublisherParams struct {
	PubSub           pubsub.Publisher[models.NodeInfo]
	NodeInfoProvider models.NodeInfoProvider
	IntervalConfig   NodeInfoPublisherIntervalConfig
}

type NodeInfoPublisher struct {
	pubSub           pubsub.Publisher[models.NodeInfo]
	nodeInfoProvider models.NodeInfoProvider
	intervalConfig   NodeInfoPublisherIntervalConfig
	stopped          bool
	stopChannel      chan struct{}
	stopOnce         sync.Once
}

func NewNodeInfoPublisher(params NodeInfoPublisherParams) *NodeInfoPublisher {
	p := &NodeInfoPublisher{
		pubSub:           params.PubSub,
		nodeInfoProvider: params.NodeInfoProvider,
		intervalConfig:   params.IntervalConfig,
		stopChannel:      make(chan struct{}),
	}

	return p
}

func (n *NodeInfoPublisher) Start(ctx context.Context) {
	go func() {
		if n.intervalConfig.IsEagerPublishEnabled() {
			n.eagerPublishBackgroundTask()
		} else {
			n.standardPublishBackgroundTask()
		}
	}()
}

// Publish publishes the node info to the pubsub topic manually and won't wait for the background task to do it.
func (n *NodeInfoPublisher) Publish(ctx context.Context) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/routing.NodeInfoPublisher.publish")
	defer span.End()

	return n.pubSub.Publish(ctx, n.nodeInfoProvider.GetNodeInfo(ctx))
}

func (n *NodeInfoPublisher) eagerPublishBackgroundTask() {
	ctx, cancel := context.WithTimeout(context.Background(), n.intervalConfig.EagerPublishDuration)
	log.Ctx(ctx).Debug().Msgf("Starting eager publish background task with interval %v for %v",
		n.intervalConfig.EagerPublishInterval, n.intervalConfig.EagerPublishDuration)
	n.publishBackgroundTask(ctx, n.intervalConfig.EagerPublishInterval)
	cancel()

	// start standard publish background task after eager publish if it wasn't stopped
	if !n.stopped {
		log.Ctx(ctx).Debug().Msgf("Starting standard publish background task with interval %v", n.intervalConfig.Interval)
		n.standardPublishBackgroundTask()
	}
}

func (n *NodeInfoPublisher) standardPublishBackgroundTask() {
	n.publishBackgroundTask(context.Background(), n.intervalConfig.Interval)
}

func (n *NodeInfoPublisher) publishBackgroundTask(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
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
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the background task that publishes the node info periodically
func (n *NodeInfoPublisher) Stop(ctx context.Context) {
	n.stopOnce.Do(func() {
		n.stopped = true
		n.stopChannel <- struct{}{}
	})
}
