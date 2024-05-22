package kvstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

const (
	V130BucketName    = "nodes"
	DefaultBucketName = "node_states"
)

type NodeStoreParams struct {
	BucketName string
	Client     *nats.Conn
}

type NodeStore struct {
	js jetstream.JetStream
	kv jetstream.KeyValue
}

func NewNodeStore(ctx context.Context, params NodeStoreParams) (*NodeStore, error) {
	js, err := jetstream.New(params.Client)
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to connect to jetstream")
	}

	bucketName := strings.ToLower(params.BucketName)
	if bucketName == "" {
		return nil, pkgerrors.New("bucket name is required")
	}

	if err := migrateNodeInfoToNodeState(ctx, js, V130BucketName, bucketName); err != nil {
		return nil, err
	}

	kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: bucketName,
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "failed to create key-value store")
	}

	return &NodeStore{
		js: js,
		kv: kv,
	}, nil
}

func migrateNodeInfoToNodeState(ctx context.Context, js jetstream.JetStream, from string, to string) (retErr error) {
	defer func() {
		if retErr == nil {
			if err := js.DeleteKeyValue(ctx, from); err != nil {
				if errors.Is(err, jetstream.ErrBucketNotFound) {
					// migration is successful since there isn't previous state to migrate from
					retErr = nil
				} else {
					retErr = fmt.Errorf("NodeStore migration succeeded, but failed to remove old bucket: %w", err)
				}
			}
		}
	}()

	fromKV, err := js.KeyValue(ctx, from)
	if err != nil {
		if errors.Is(err, jetstream.ErrBucketNotFound) {
			// migration is successful since there isn't previous state to migrate from
			return nil
		}
		return fmt.Errorf("NodeStore migration failed: failed to open 'from' bucket: %w", err)
	}

	keys, err := fromKV.Keys(ctx)
	if err != nil {
		if pkgerrors.Is(err, jetstream.ErrNoKeysFound) {
			// if the store is empty the migration is successful as there isn't anything to migrate
			return nil
		}
		return fmt.Errorf("NodeStore migration failed: failed to list store: %w", err)
	}

	nodeInfos := make([]models.NodeInfo, 0, len(keys))
	for _, key := range keys {
		entry, err := fromKV.Get(ctx, key)
		if err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to read node info with name: %s: %w", key, err)
		}

		var nodeinfo models.NodeInfo
		if err := json.Unmarshal(entry.Value(), &nodeinfo); err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to unmarshal node info: %w", err)
		}
		nodeInfos = append(nodeInfos, nodeinfo)
	}

	toKV, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: to,
	})
	if err != nil {
		return fmt.Errorf("NodeStore migration failed: failed to open to bucket: %w", err)
	}

	for _, ni := range nodeInfos {
		nodestate := models.NodeState{
			Info:       ni,
			Membership: models.NodeMembership.PENDING,
			Connection: models.NodeStates.DISCONNECTED,
		}
		data, err := json.Marshal(nodestate)
		if err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to marshal node state: %w", err)
		}
		if _, err := toKV.Put(ctx, nodestate.Info.ID(), data); err != nil {
			return fmt.Errorf("NodeStore migration failed: failed to write node state to store: %w", err)
		}
	}

	return nil
}

func (n *NodeStore) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	// TODO: Remove this once we now longer need to implement the routing.PeerStore interface
	// We are temporarily matching the code of the inmemory.NodeStore which never returns an
	// error for this method.
	nodeID := peerID.String()
	state, err := n.Get(ctx, nodeID)
	if err != nil {
		return peer.AddrInfo{}, nil
	}

	return *state.Info.PeerInfo, nil
}

// Add adds a node state to the repo.
func (n *NodeStore) Add(ctx context.Context, state models.NodeState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to marshal node state adding to node store")
	}

	_, err = n.kv.Put(ctx, state.Info.ID(), data)
	if err != nil {
		return pkgerrors.Wrap(err, "failed to write node state to node store")
	}

	return nil
}

// Get returns the node state for the given node ID.
func (n *NodeStore) Get(ctx context.Context, nodeID string) (models.NodeState, error) {
	entry, err := n.kv.Get(ctx, nodeID)
	if err != nil {
		if pkgerrors.Is(err, jetstream.ErrKeyNotFound) {
			return models.NodeState{}, routing.NewErrNodeNotFound(nodeID)
		}

		return models.NodeState{}, pkgerrors.Wrap(err, "failed to get node state from node store")
	}

	var node models.NodeState
	err = json.Unmarshal(entry.Value(), &node)
	if err != nil {
		return models.NodeState{}, pkgerrors.Wrap(err, "failed to unmarshal node state from node store")
	}

	return node, nil
}

// GetByPrefix returns the node state for the given node ID.
// Supports both full and short node IDs band currently iterates through all of the
// keys to find matches, due to NATS KVStore not supporting prefix searches (yet).
func (n *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeState, error) {
	keys, err := n.kv.Keys(ctx)
	if err != nil {
		if pkgerrors.Is(err, jetstream.ErrNoKeysFound) {
			return models.NodeState{}, routing.NewErrNodeNotFound(prefix)
		}
		return models.NodeState{}, pkgerrors.Wrap(err, "failed to get by prefix when listing keys")
	}

	// Filter the list down to just the matching keys
	keys = lo.Filter(keys, func(item string, index int) bool {
		return strings.HasPrefix(item, prefix)
	})

	if len(keys) == 0 {
		return models.NodeState{}, routing.NewErrNodeNotFound(prefix)
	} else if len(keys) > 1 {
		return models.NodeState{}, routing.NewErrMultipleNodesFound(prefix, keys)
	}

	return n.Get(ctx, keys[0])
}

// List returns a list of nodes
func (n *NodeStore) List(ctx context.Context, filters ...routing.NodeStateFilter) ([]models.NodeState, error) {
	keys, err := n.kv.Keys(ctx)
	if err != nil {
		// Return an empty list rather than an error if there are no keys in the bucket
		if pkgerrors.Is(err, jetstream.ErrNoKeysFound) {
			return []models.NodeState{}, nil
		}
		return nil, pkgerrors.Wrap(err, "failed to list node state from node store")
	}

	var mErr error

	// Create a mega filter that combines all the filters into one
	megaFilter := func(state models.NodeState) bool {
		for _, filter := range filters {
			if !filter(state) {
				return false
			}
		}
		return true
	}

	nodes := make([]models.NodeState, 0, len(keys))
	for _, key := range keys {
		node, err := n.Get(ctx, key)
		if err != nil {
			mErr = errors.Join(mErr, err)
		}

		if megaFilter(node) {
			nodes = append(nodes, node)
		}
	}

	return nodes, mErr
}

// Delete deletes a node state from the repo.
func (n *NodeStore) Delete(ctx context.Context, nodeID string) error {
	if err := n.kv.Purge(ctx, nodeID); err != nil {
		return pkgerrors.Wrap(err, "failed to purge node state from node store")
	}

	return nil
}

var _ routing.NodeInfoStore = (*NodeStore)(nil)
