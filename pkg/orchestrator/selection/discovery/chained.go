package discovery

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
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
	uniqueNodes := make(map[string]models.NodeInfo, 0)
	for _, discoverer := range c.discoverers {
		nodeInfos, discoverErr := getNodes(discoverer)
		err = errors.Join(err, pkgerrors.Wrapf(discoverErr, "error finding nodes from %T", discoverer))
		currentNodesCount := len(uniqueNodes)
		for _, nodeInfo := range nodeInfos {
			if _, ok := uniqueNodes[nodeInfo.ID()]; !ok {
				uniqueNodes[nodeInfo.ID()] = nodeInfo
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
