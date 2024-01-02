package discovery

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
	"golang.org/x/exp/maps"
)

type Chain struct {
	discoverers  []orchestrator.NodeDiscoverer
	ignoreErrors bool
}

func NewChain(ignoreErrors bool) *Chain {
	return &Chain{
		ignoreErrors: ignoreErrors,
	}
}

func (c *Chain) Add(discoverer ...orchestrator.NodeDiscoverer) {
	c.discoverers = append(c.discoverers, discoverer...)
}

func (c *Chain) FindNodes(ctx context.Context, job models.Job) ([]models.NodeInfo, error) {
	return c.chainDiscovery(ctx, "FindNodes", func(r orchestrator.NodeDiscoverer) ([]models.NodeInfo, error) {
		return r.FindNodes(ctx, job)
	})
}

func (c *Chain) ListNodes(ctx context.Context) ([]models.NodeInfo, error) {
	return c.chainDiscovery(ctx, "ListNodes", func(r orchestrator.NodeDiscoverer) ([]models.NodeInfo, error) {
		return r.ListNodes(ctx)
	})
}

func (c *Chain) chainDiscovery(
	ctx context.Context,
	caller string,
	getNodes func(orchestrator.NodeDiscoverer) ([]models.NodeInfo, error),
) ([]models.NodeInfo, error) {
	var err error
	uniqueNodes := make(map[peer.ID]models.NodeInfo, 0)
	for _, discoverer := range c.discoverers {
		nodeInfos, discoverErr := getNodes(discoverer)
		err = multierr.Append(err, errors.Wrapf(discoverErr, "error finding nodes from %T", discoverer))
		currentNodesCount := len(uniqueNodes)
		for _, nodeInfo := range nodeInfos {
			if _, ok := uniqueNodes[nodeInfo.PeerInfo.ID]; !ok {
				uniqueNodes[nodeInfo.PeerInfo.ID] = nodeInfo
			}
		}
		log.Ctx(ctx).Debug().Msgf("[%s] found %d more nodes by %T", caller, len(uniqueNodes)-currentNodesCount, discoverer)
	}

	if err != nil && c.ignoreErrors {
		log.Ctx(ctx).Warn().Err(err).Msg("ignoring error finding nodes")
		err = nil
	}

	return maps.Values(uniqueNodes), err
}

// compile-time interface assertions
var _ orchestrator.NodeDiscoverer = (*Chain)(nil)
