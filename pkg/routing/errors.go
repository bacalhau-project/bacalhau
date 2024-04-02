package routing

import (
	"fmt"
)

// ErrNodeNotFound is returned when nodeInfo was not found for a requested node id
type ErrNodeNotFound struct {
	nodeID string
}

func NewErrNodeNotFound(nodeID string) ErrNodeNotFound {
	return ErrNodeNotFound{nodeID: nodeID}
}

func (e ErrNodeNotFound) Error() string {
	return fmt.Errorf("nodeInfo not found for nodeID: %s", e.nodeID).Error()
}

// ErrMultipleNodesFound is returned when multiple nodes were found for a requested node id prefix
type ErrMultipleNodesFound struct {
	nodeIDPrefix    string
	matchingNodeIDs []string
}

func NewErrMultipleNodesFound(nodeIDPrefix string, matchingNodeIDs []string) ErrMultipleNodesFound {
	if len(matchingNodeIDs) > 3 {
		matchingNodeIDs = append(matchingNodeIDs[:3], "...")
	}
	return ErrMultipleNodesFound{nodeIDPrefix: nodeIDPrefix, matchingNodeIDs: matchingNodeIDs}
}

func (e ErrMultipleNodesFound) Error() string {
	return fmt.Errorf("multiple nodes found for nodeID prefix: %s, matching nodeIDs: %v", e.nodeIDPrefix, e.matchingNodeIDs).Error()
}
