package discovery

import (
	"context"
	"reflect"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
)

type Chained struct {
	discoverers  []requester.NodeDiscoverer
	ignoreErrors bool
}

func NewChained(ignoreErrors bool) *Chained {
	return &Chained{
		ignoreErrors: ignoreErrors,
	}
}

func (c *Chained) Add(discoverer ...requester.NodeDiscoverer) {
	c.discoverers = append(c.discoverers, discoverer...)
}

func (c *Chained) FindNodes(ctx context.Context, job model.Job) ([]peer.ID, error) {
	uniqueNodes := make(map[peer.ID]peer.ID, 0)
	for _, discoverer := range c.discoverers {
		peerIDs, err := discoverer.FindNodes(ctx, job)
		if err != nil {
			if !c.ignoreErrors {
				return nil, err
			} else {
				log.Ctx(ctx).Warn().Err(err).Msgf("ignoring error finding nodes by %s", reflect.TypeOf(discoverer))
			}
		}
		currentNodesCount := len(uniqueNodes)
		for _, peerID := range peerIDs {
			uniqueNodes[peerID] = peerID
		}
		log.Debug().Msgf("found %d more nodes by %s", len(uniqueNodes)-currentNodesCount, reflect.TypeOf(discoverer))
	}
	peerIDs := make([]peer.ID, 0, len(uniqueNodes))
	for _, peerID := range uniqueNodes {
		peerIDs = append(peerIDs, peerID)
	}
	return peerIDs, nil
}

// compile-time interface assertions
var _ requester.NodeDiscoverer = (*Chained)(nil)
