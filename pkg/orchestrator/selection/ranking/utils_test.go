package ranking

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

func assertEquals(t *testing.T, ranks []orchestrator.NodeRank, nodeID string, expectedRank int) {
	for _, rank := range ranks {
		if rank.NodeInfo.ID() == nodeID {
			if rank.Rank != expectedRank {
				t.Errorf("expected rank %d for node %s, got %d", expectedRank, nodeID, rank.Rank)
			}
			return
		}
	}
	t.Errorf("node %s not found", nodeID)
}
