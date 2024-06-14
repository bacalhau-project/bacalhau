package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

// TODO: replace the manual and lazy eviction with a more efficient caching library
type nodeInfoWrapper struct {
	models.NodeState
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

func (r *NodeStore) Add(ctx context.Context, state models.NodeState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// add or update the node info
	nodeID := state.Info.ID()
	r.nodeInfoMap[nodeID] = nodeInfoWrapper{
		NodeState: state,
		evictAt:   time.Now().Add(r.ttl),
	}

	log.Ctx(ctx).Trace().Msgf("Added node state %+v", state)
	return nil
}

func (r *NodeStore) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	infoWrapper, ok := r.nodeInfoMap[nodeID]
	if !ok {
		return models.NodeState{}, routing.NewErrNodeNotFound(nodeID)
	}
	if time.Now().After(infoWrapper.evictAt) {
		go r.evict(ctx, infoWrapper)
		return models.NodeState{}, routing.NewErrNodeNotFound(nodeID)
	}
	return infoWrapper.NodeState, nil
}

func (r *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, err := r.Get(ctx, prefix)

	// we found a node with the exact ID
	if err == nil {
		return state, nil
	}
	// return the error if it's not a node not found error
	var errNotFound routing.ErrNodeNotFound
	if !errors.As(err, &errNotFound) {
		return models.NodeState{}, err
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
		return models.NodeState{}, routing.NewErrNodeNotFound(prefix)
	}

	if len(nodeIDsWithPrefix) > 1 {
		return models.NodeState{}, routing.NewErrMultipleNodesFound(prefix, nodeIDsWithPrefix)
	}

	return r.nodeInfoMap[nodeIDsWithPrefix[0]].NodeState, nil
}

func (r *NodeStore) List(ctx context.Context, filters ...routing.NodeStateFilter) ([]models.NodeState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	megaFilter := func(state models.NodeState) bool {
		for _, filter := range filters {
			if !filter(state) {
				return false
			}
		}
		return true
	}

	var nodeStates []models.NodeState
	var toEvict []nodeInfoWrapper
	for _, nodeState := range r.nodeInfoMap {
		if time.Now().After(nodeState.evictAt) {
			toEvict = append(toEvict, nodeState)
		} else {
			if megaFilter(nodeState.NodeState) {
				nodeStates = append(nodeStates, nodeState.NodeState)
			}
		}
	}
	if len(toEvict) > 0 {
		go r.evict(ctx, toEvict...)
	}
	return nodeStates, nil
}

func (r *NodeStore) Delete(ctx context.Context, nodeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.doDelete(ctx, nodeID)
}

func (r *NodeStore) evict(ctx context.Context, stateWrappers ...nodeInfoWrapper) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, stateWrapper := range stateWrappers {
		nodeID := stateWrapper.Info.ID()
		nodeInfo, ok := r.nodeInfoMap[nodeID]
		if !ok || nodeInfo.evictAt != stateWrapper.evictAt {
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
