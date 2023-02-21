package discovery

import (
	"context"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
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
	uniqueNodes := make(map[peer.ID]model.NodeInfo, 0)
	for _, discoverer := range c.discoverers {
		nodeInfos, err := discoverer.FindNodes(ctx, job)
		if err != nil {
			if !c.ignoreErrors {
				return nil, err
			} else {
				log.Ctx(ctx).Warn().Err(err).Msgf("ignoring error finding nodes by %s", reflect.TypeOf(discoverer))
			}
		}
		currentNodesCount := len(uniqueNodes)
		for _, nodeInfo := range nodeInfos {
			if _, ok := uniqueNodes[nodeInfo.PeerInfo.ID]; !ok {
				uniqueNodes[nodeInfo.PeerInfo.ID] = nodeInfo
			}
		}
		log.Ctx(ctx).Debug().Msgf("found %d more nodes by %s", len(uniqueNodes)-currentNodesCount, reflect.TypeOf(discoverer))
	}
	nodeInfos := make([]model.NodeInfo, 0, len(uniqueNodes))
	for _, nodeInfo := range uniqueNodes {
		nodeInfos = append(nodeInfos, nodeInfo)
	}
	return nodeInfos, nil
}

// compile-time interface assertions
var _ requester.NodeDiscoverer = (*Chain)(nil)
