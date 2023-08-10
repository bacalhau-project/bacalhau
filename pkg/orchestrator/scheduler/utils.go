package scheduler

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
)

// existingNodeInfos returns a map of nodeID to NodeInfo for all the nodes that have executions for this job
func existingNodeInfos(
	ctx context.Context, nodeDiscoverer orchestrator.NodeDiscoverer, existingExecutions execSet) (map[string]*models.NodeInfo, error) {
	out := make(map[string]*models.NodeInfo)
	if len(existingExecutions) == 0 {
		return out, nil
	}
	checked := make(map[string]struct{})

	// TODO: implement a better way to retrieve node info instead of listing all nodes
	nodesMap := make(map[string]*models.NodeInfo)
	discoveredNodes, err := nodeDiscoverer.ListNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}
	for i, node := range discoveredNodes {
		nodesMap[node.PeerInfo.ID.String()] = &discoveredNodes[i]
	}

	for _, execution := range existingExecutions {
		// keep track of the nodes that we already checked, including the nodes
		// that no longer exist in the node discoverer
		if _, ok := checked[execution.NodeID]; ok {
			continue
		}
		nodeInfo, ok := nodesMap[execution.NodeID]
		if ok {
			out[execution.NodeID] = nodeInfo
		}
		checked[execution.NodeID] = struct{}{}
	}
	return out, nil
}
