package kvstore

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	pkgerrors "github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
)

const (
	// BucketNameCurrent is the bucket name for bacalhau version v1.3.1 and beyond.
	BucketNameCurrent = "node_v1"
	// BucketNameV0 is the bucket name for bacalhau version v1.3.0 and below.
	BucketNameV0 = "nodes"
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

// Put adds a node state to the repo.
func (n *NodeStore) Put(ctx context.Context, state models.NodeState) error {
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
			return models.NodeState{}, nodes.NewErrNodeNotFound(nodeID)
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
			return models.NodeState{}, nodes.NewErrNodeNotFound(prefix)
		}
		return models.NodeState{}, pkgerrors.Wrap(err, "failed to get by prefix when listing keys")
	}

	// Filter the list down to just the matching keys
	keys = lo.Filter(keys, func(item string, index int) bool {
		return strings.HasPrefix(item, prefix)
	})

	if len(keys) == 0 {
		return models.NodeState{}, nodes.NewErrNodeNotFound(prefix)
	} else if len(keys) > 1 {
		return models.NodeState{}, nodes.NewErrMultipleNodesFound(prefix, keys)
	}

	return n.Get(ctx, keys[0])
}

// List returns a list of nodes
func (n *NodeStore) List(ctx context.Context, filters ...nodes.NodeStateFilter) ([]models.NodeState, error) {
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

var _ nodes.Store = (*NodeStore)(nil)
