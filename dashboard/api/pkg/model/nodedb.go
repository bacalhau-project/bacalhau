package model

import (
	"context"
	"sync"
	"time"

	bacalhau_model "github.com/filecoin-project/bacalhau/pkg/model"
)

type ClusterMapNode struct {
	ID    string `json:"id"`
	Group int    `json:"group"`
}

type ClusterMapLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type ClusterMapResult struct {
	Nodes []ClusterMapNode `json:"nodes"`
	Links []ClusterMapLink `json:"links"`
}

type nodeDB struct {
	nodes map[string]bacalhau_model.NodeEvent
	mtx   sync.Mutex
}

const CleanupLoopInterval = time.Second * 10

func newNodeDB() (*nodeDB, error) {
	nodeDB := &nodeDB{
		nodes: make(map[string]bacalhau_model.NodeEvent),
	}
	return nodeDB, nil
}

func (db *nodeDB) getNodes() map[string]bacalhau_model.NodeEvent {
	return db.nodes
}

func (db *nodeDB) getClusterMap() ClusterMapResult {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	data := ClusterMapResult{
		Nodes: []ClusterMapNode{},
		Links: []ClusterMapLink{},
	}
	for nodeID, nodeEvent := range db.nodes {
		data.Nodes = append(data.Nodes, ClusterMapNode{
			ID:    nodeID,
			Group: 0,
		})
		topics := nodeEvent.Peers
		for _, peers := range topics {
			for _, peer := range peers {
				data.Links = append(data.Links, ClusterMapLink{
					Source: nodeID,
					Target: peer.String(),
				})
			}
			break
		}
	}
	return data
}

func (db *nodeDB) addEvent(event bacalhau_model.NodeEvent) {
	db.mtx.Lock()
	defer db.mtx.Unlock()
	db.nodes[event.NodeID] = event
}

// cleanup nodes we have not heard about for a minute
func (db *nodeDB) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(CleanupLoopInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			db.cleanup()
		}
	}
}

func (db *nodeDB) cleanup() {
	// loop over nodes and check events EventTime to see if
	// it's more than a minute old and remove from nodes map
	db.mtx.Lock()
	defer db.mtx.Unlock()
	for nodeID, event := range db.nodes {
		if time.Since(event.EventTime) > 1*time.Minute {
			delete(db.nodes, nodeID)
		}
	}
}
