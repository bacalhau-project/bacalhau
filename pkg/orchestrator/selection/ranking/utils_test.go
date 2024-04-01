package ranking

import (
	"slices"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

func assertEquals(t *testing.T, ranks []orchestrator.NodeRank, nodeID string, expectedRank int, expectedReason ...string) {
	for _, rank := range ranks {
		if rank.NodeInfo.ID() == nodeID {
			if rank.Rank != expectedRank {
				t.Errorf("expected rank %d for node %s, got %d", expectedRank, nodeID, rank.Rank)
			}
			if len(expectedReason) > 0 && !slices.Contains(expectedReason, rank.Reason) {
				t.Errorf("expected reason %q for node %s, got %q", expectedReason[0], nodeID, rank.Reason)
			}
			return
		}
	}
	t.Errorf("node %s not found", nodeID)
}
