package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

type Chain struct {
	discoverers  []requester.NodeDiscoverer
	ignoreErrors bool
}

func NewChain(ignoreErrors bool) *Chain {
	return &Chain{
		ignoreErrors: ignoreErrors,
	}
}

func (c *Chain) Add(discoverer ...requester.NodeDiscoverer) {
	c.discoverers = append(c.discoverers, discoverer...)
}

func (c *Chain) FindNodes(ctx context.Context, job model.Job) ([]model.NodeInfo, error) {
	return c.ChainDiscovery(ctx, func(r requester.NodeDiscoverer) ([]model.NodeInfo, error) { return r.FindNodes(ctx, job) })
}

func (c *Chain) ListNodes(ctx context.Context) ([]model.NodeInfo, error) {
	return c.ChainDiscovery(ctx, func(r requester.NodeDiscoverer) ([]model.NodeInfo, error) { return r.ListNodes(ctx) })
}

func (c *Chain) ChainDiscovery(
	ctx context.Context,
	getNodes func(requester.NodeDiscoverer) ([]model.NodeInfo, error),
) ([]model.NodeInfo, error) {
	var err error
	uniqueNodes := make(map[peer.ID]model.NodeInfo, 0)
	for _, discoverer := range c.discoverers {
		nodeInfos, discoverErr := getNodes(discoverer)
		err = multierr.Append(err, errors.Wrapf(discoverErr, "error finding nodes from %T", discoverer))
		currentNodesCount := len(uniqueNodes)
		for _, nodeInfo := range nodeInfos {
			if _, ok := uniqueNodes[nodeInfo.PeerInfo.ID]; !ok {
				uniqueNodes[nodeInfo.PeerInfo.ID] = nodeInfo
			}
		}
		log.Ctx(ctx).Debug().Msgf("found %d more nodes by %T", len(uniqueNodes)-currentNodesCount, discoverer)
	}

	if err != nil && c.ignoreErrors {
		log.Ctx(ctx).Warn().Err(err).Msg("ignoring error finding nodes")
		err = nil
	}

	return maps.Values(uniqueNodes), err
}

// compile-time interface assertions
var _ requester.NodeDiscoverer = (*Chain)(nil)
