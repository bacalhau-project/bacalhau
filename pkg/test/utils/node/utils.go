package node

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
)

// WaitForNodeDiscovery for the requester node to pick up the nodeInfo messages
func WaitForNodeDiscovery(t *testing.T, requesterNode *node.Node, expectedNodeCount int) {
	ctx := context.Background()
	waitDuration := 15 * time.Second
	waitGaps := 20 * time.Millisecond
	waitUntil := time.Now().Add(waitDuration)
	loggingGap := 1 * time.Second
	waitLoggingUntil := time.Now().Add(loggingGap)

	var nodeInfos []models.NodeInfo
	for time.Now().Before(waitUntil) {
		var err error
		nodeInfos, err = requesterNode.NodeInfoStore.List(ctx)
		require.NoError(t, err)
		if time.Now().After(waitLoggingUntil) {
			t.Logf("connected to %d peers: %v", len(nodeInfos), logger.ToSliceStringer(nodeInfos, func(t models.NodeInfo) string {
				return t.ID()
			}))
			waitLoggingUntil = time.Now().Add(loggingGap)
		}
		if len(nodeInfos) == expectedNodeCount {
			return
		}
		time.Sleep(waitGaps)
	}
	require.FailNowf(t, fmt.Sprintf("requester node didn't read all node infos even after waiting for %s", waitDuration),
		"expected 4 node infos, got %d. %+v", len(nodeInfos), logger.ToSliceStringer(nodeInfos, func(t models.NodeInfo) string {
			return t.ID()
		}))
}
