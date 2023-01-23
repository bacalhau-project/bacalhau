package ranking

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/peer"
)

func assertEquals(t *testing.T, ranks []requester.NodeRank, nodeID string, expectedRank int) {
	for _, rank := range ranks {
		if rank.NodeInfo.PeerInfo.ID == peer.ID(nodeID) {
			if rank.Rank != expectedRank {
				t.Errorf("expected rank %d for node %s, got %d", expectedRank, nodeID, rank.Rank)
			}
			return
		}
	}
	t.Errorf("node %s not found", nodeID)
}
