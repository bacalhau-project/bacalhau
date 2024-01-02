package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

// TODO: replace the manual and lazy eviction with a more efficient caching library
type nodeInfoWrapper struct {
	models.NodeInfo
	evictAt time.Time
}

type NodeInfoStoreParams struct {
	TTL time.Duration
}

type NodeInfoStore struct {
	ttl             time.Duration
	nodeInfoMap     map[string]nodeInfoWrapper
	engineNodeIDMap map[string]map[string]struct{}
	mu              sync.RWMutex
}

func NewNodeInfoStore(params NodeInfoStoreParams) *NodeInfoStore {
	return &NodeInfoStore{
		ttl:             params.TTL,
		nodeInfoMap:     make(map[string]nodeInfoWrapper),
		engineNodeIDMap: make(map[string]map[string]struct{}),
	}
}

func (r *NodeInfoStore) Add(ctx context.Context, nodeInfo models.NodeInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// delete node from previous engines if it already exists to replace old engines with new ones if they've changed
	nodeID := nodeInfo.ID()
	existingNodeInfo, ok := r.nodeInfoMap[nodeID]
	if ok {
		if existingNodeInfo.ComputeNodeInfo != nil {
			for _, engine := range existingNodeInfo.ComputeNodeInfo.ExecutionEngines {
				delete(r.engineNodeIDMap[engine], nodeID)
			}
		}
	} else {
		var engines []string
		if nodeInfo.ComputeNodeInfo != nil {
			engines = append(engines, nodeInfo.ComputeNodeInfo.ExecutionEngines...)
		}
		log.Ctx(ctx).Debug().Msgf("Adding new node %s to in-memory nodeInfo store with engines %v", nodeID, engines)
	}

	// TODO: use data structure that maintains nodes in descending order based on available capacity.
	if nodeInfo.ComputeNodeInfo != nil {
		for _, engine := range nodeInfo.ComputeNodeInfo.ExecutionEngines {
			if _, ok := r.engineNodeIDMap[engine]; !ok {
				r.engineNodeIDMap[engine] = make(map[string]struct{})
			}
			r.engineNodeIDMap[engine][nodeID] = struct{}{}
		}
	}

	// add or update the node info
	r.nodeInfoMap[nodeID] = nodeInfoWrapper{
		NodeInfo: nodeInfo,
		evictAt:  time.Now().Add(r.ttl),
	}

	log.Ctx(ctx).Trace().Msgf("Added node info %+v", nodeInfo)
	return nil
}

func (r *NodeInfoStore) Get(ctx context.Context, nodeID string) (models.NodeInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infoWrapper, ok := r.nodeInfoMap[nodeID]
	if !ok {
		return models.NodeInfo{}, routing.NewErrNodeNotFound(nodeID)
	}
	if time.Now().After(infoWrapper.evictAt) {
		go r.evict(ctx, infoWrapper)
		return models.NodeInfo{}, routing.NewErrNodeNotFound(nodeID)
	}
	return infoWrapper.NodeInfo, nil
}

func (r *NodeInfoStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nodeInfo, err := r.Get(ctx, prefix)

	// we found a node with the exact ID
	if err == nil {
		return nodeInfo, nil
	}
	// return the error if it's not a node not found error
	var errNotFound routing.ErrNodeNotFound
	if !errors.As(err, &errNotFound) {
		return models.NodeInfo{}, err
	}

	// look for a node with the prefix. if there are multiple nodes with the same prefix, return ErrMultipleNodesFound error
	var nodeIDsWithPrefix []string
	var toEvict []nodeInfoWrapper
	for nodeID, infoWrapper := range r.nodeInfoMap {
		if time.Now().After(infoWrapper.evictAt) {
			toEvict = append(toEvict, infoWrapper)
		} else if nodeID[:len(prefix)] == prefix {
			nodeIDsWithPrefix = append(nodeIDsWithPrefix, nodeID)
		}
	}

	if len(toEvict) > 0 {
		go r.evict(ctx, toEvict...)
	}

	if len(nodeIDsWithPrefix) == 0 {
		return models.NodeInfo{}, routing.NewErrNodeNotFound(prefix)
	}

	if len(nodeIDsWithPrefix) > 1 {
		return models.NodeInfo{}, routing.NewErrMultipleNodesFound(prefix, nodeIDsWithPrefix)
	}

	return r.nodeInfoMap[nodeIDsWithPrefix[0]].NodeInfo, nil
}

func (r *NodeInfoStore) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infoWrapper, ok := r.nodeInfoMap[peerID.String()]
	if !ok {
		return peer.AddrInfo{}, nil
	}
	if len(infoWrapper.PeerInfo.Addrs) > 0 {
		return infoWrapper.PeerInfo, nil
	}
	return peer.AddrInfo{}, nil
}

func (r *NodeInfoStore) List(ctx context.Context) ([]models.NodeInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var nodeInfos []models.NodeInfo
	var toEvict []nodeInfoWrapper
	for _, nodeInfo := range r.nodeInfoMap {
		if time.Now().After(nodeInfo.evictAt) {
			toEvict = append(toEvict, nodeInfo)
		} else {
			nodeInfos = append(nodeInfos, nodeInfo.NodeInfo)
		}
	}
	if len(toEvict) > 0 {
		go r.evict(ctx, toEvict...)
	}
	return nodeInfos, nil
}

func (r *NodeInfoStore) ListForEngine(ctx context.Context, engine string) ([]models.NodeInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var nodeInfos []models.NodeInfo
	var toEvict []nodeInfoWrapper
	for nodeID := range r.engineNodeIDMap[engine] {
		nodeInfo := r.nodeInfoMap[nodeID]
		if time.Now().After(nodeInfo.evictAt) {
			toEvict = append(toEvict, nodeInfo)
		} else {
			nodeInfos = append(nodeInfos, nodeInfo.NodeInfo)
		}
	}
	if len(toEvict) > 0 {
		go r.evict(ctx, toEvict...)
	}
	return nodeInfos, nil
}

func (r *NodeInfoStore) Delete(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.doDelete(ctx, nodeID)
}

func (r *NodeInfoStore) evict(ctx context.Context, infoWrappers ...nodeInfoWrapper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, infoWrapper := range infoWrappers {
		nodeID := infoWrapper.ID()
		nodeInfo, ok := r.nodeInfoMap[nodeID]
		if !ok || nodeInfo.evictAt != infoWrapper.evictAt {
			return // node info already evicted or has been updated since it was scheduled for eviction
		}
		err := r.doDelete(ctx, nodeID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("Failed to evict expired node info for peer %s", nodeID)
		}
	}
}

func (r *NodeInfoStore) doDelete(ctx context.Context, nodeID string) error {
	nodeInfo, ok := r.nodeInfoMap[nodeID]
	if !ok {
		return nil
	}
	for _, engine := range nodeInfo.ComputeNodeInfo.ExecutionEngines {
		delete(r.engineNodeIDMap[engine], nodeID)
	}
	delete(r.nodeInfoMap, nodeID)
	return nil
}

// compile time check that we implement the interface
var _ routing.NodeInfoStore = (*NodeInfoStore)(nil)
