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

type NodeStoreParams struct {
	TTL time.Duration
}

type NodeStore struct {
	ttl         time.Duration
	nodeInfoMap map[string]nodeInfoWrapper
	mu          sync.RWMutex
}

func NewNodeStore(params NodeStoreParams) *NodeStore {
	return &NodeStore{
		ttl:         params.TTL,
		nodeInfoMap: make(map[string]nodeInfoWrapper),
	}
}

func (r *NodeStore) Add(ctx context.Context, nodeInfo models.NodeInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// add or update the node info
	nodeID := nodeInfo.ID()
	r.nodeInfoMap[nodeID] = nodeInfoWrapper{
		NodeInfo: nodeInfo,
		evictAt:  time.Now().Add(r.ttl),
	}

	log.Ctx(ctx).Trace().Msgf("Added node info %+v", nodeInfo)
	return nil
}

func (r *NodeStore) Get(ctx context.Context, nodeID string) (models.NodeInfo, error) {
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

func (r *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeInfo, error) {
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

func (r *NodeStore) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infoWrapper, ok := r.nodeInfoMap[peerID.String()]
	if !ok {
		return peer.AddrInfo{}, nil
	}
	if infoWrapper.PeerInfo != nil && len(infoWrapper.PeerInfo.Addrs) > 0 {
		return *infoWrapper.PeerInfo, nil
	}
	return peer.AddrInfo{}, nil
}

func (r *NodeStore) List(ctx context.Context, filters ...routing.NodeInfoFilter) ([]models.NodeInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	megaFilter := func(info models.NodeInfo) bool {
		for _, filter := range filters {
			if !filter(info) {
				return false
			}
		}
		return true
	}

	var nodeInfos []models.NodeInfo
	var toEvict []nodeInfoWrapper
	for _, nodeInfo := range r.nodeInfoMap {
		if time.Now().After(nodeInfo.evictAt) {
			toEvict = append(toEvict, nodeInfo)
		} else {
			if megaFilter(nodeInfo.NodeInfo) {
				nodeInfos = append(nodeInfos, nodeInfo.NodeInfo)
			}
		}
	}
	if len(toEvict) > 0 {
		go r.evict(ctx, toEvict...)
	}
	return nodeInfos, nil
}

func (r *NodeStore) Delete(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.doDelete(ctx, nodeID)
}

func (r *NodeStore) evict(ctx context.Context, infoWrappers ...nodeInfoWrapper) {
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

func (r *NodeStore) doDelete(ctx context.Context, nodeID string) error {
	delete(r.nodeInfoMap, nodeID)
	return nil
}

// compile time check that we implement the interface
var _ routing.NodeInfoStore = (*NodeStore)(nil)
