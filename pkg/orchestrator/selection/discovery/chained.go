package discovery

import (
	"context"
	"errors"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
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

func (c *Chain) ListNodes(ctx context.Context) ([]models.NodeState, error) {
	return c.chainDiscovery(ctx, "ListNodes", func(r orchestrator.NodeDiscoverer) ([]models.NodeState, error) {
		return r.ListNodes(ctx)
	})
}

func (c *Chain) chainDiscovery(
	ctx context.Context,
	caller string,
	getNodes func(orchestrator.NodeDiscoverer) ([]models.NodeState, error),
) ([]models.NodeState, error) {
	var (
		err         error
		uniqueNodes = make(map[string]models.NodeState, 0)
	)

	for _, discoverer := range c.discoverers {
		nodeStates, discoverErr := getNodes(discoverer)
		err = errors.Join(err, pkgerrors.Wrapf(discoverErr, "error finding nodes from %T", discoverer))
		currentNodesCount := len(uniqueNodes)
		for _, nodeState := range nodeStates {
			if _, ok := uniqueNodes[nodeState.Info.ID()]; !ok {
				uniqueNodes[nodeState.Info.ID()] = nodeState
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
