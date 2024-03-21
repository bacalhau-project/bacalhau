package kvstore

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"
	"github.com/samber/lo"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

const (
	DefaultBucketName = "nodes"
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
		return nil, errors.Wrap(err, "failed to connect to jetstream")
	}

	bucketName := strings.ToLower(params.BucketName)
	if bucketName == "" {
		return nil, errors.New("bucket name is required")
	}
	kv, err := js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
		Bucket: bucketName,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create key-value store")
	}

	return &NodeStore{
		js: js,
		kv: kv,
	}, nil
}

func (n *NodeStore) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	// TODO: Remove this once we now longer need to implement the routing.PeerStore interface
	// We are temporarily matching the code of the inmemory.NodeStore which never returns an
	// error for this method.
	nodeID := peerID.String()
	info, err := n.Get(ctx, nodeID)
	if err != nil {
		return peer.AddrInfo{}, nil
	}

	return *info.PeerInfo, nil
}

// Add adds a node info to the repo.
func (n *NodeStore) Add(ctx context.Context, nodeInfo models.NodeInfo) error {
	data, err := json.Marshal(nodeInfo)
	if err != nil {
		return errors.Wrap(err, "failed to marshal node info adding to node store")
	}

	_, err = n.kv.Put(ctx, nodeInfo.ID(), data)
	if err != nil {
		return errors.Wrap(err, "failed to write node info to node store")
	}

	return nil
}

// Get returns the node info for the given node ID.
func (n *NodeStore) Get(ctx context.Context, nodeID string) (models.NodeInfo, error) {
	entry, err := n.kv.Get(ctx, nodeID)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return models.NodeInfo{}, routing.NewErrNodeNotFound(nodeID)
		}

		return models.NodeInfo{}, errors.Wrap(err, "failed to get node info from node store")
	}

	var node models.NodeInfo
	err = json.Unmarshal(entry.Value(), &node)
	if err != nil {
		return models.NodeInfo{}, errors.Wrap(err, "failed to unmarshal node info from node store")
	}

	return node, nil
}

// GetByPrefix returns the node info for the given node ID.
// Supports both full and short node IDs band currently iterates through all of the
// keys to find matches, due to NATS KVStore not supporting prefix searches (yet).
func (n *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeInfo, error) {
	keys, err := n.kv.Keys(ctx)
	if err != nil {
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return models.NodeInfo{}, routing.NewErrNodeNotFound(prefix)
		}
		return models.NodeInfo{}, errors.Wrap(err, "failed to get by prefix when listing keys")
	}

	// Filter the list down to just the matching keys
	keys = lo.Filter(keys, func(item string, index int) bool {
		return strings.HasPrefix(item, prefix)
	})

	if len(keys) == 0 {
		return models.NodeInfo{}, routing.NewErrNodeNotFound(prefix)
	} else if len(keys) > 1 {
		return models.NodeInfo{}, routing.NewErrMultipleNodesFound(prefix, keys)
	}

	return n.Get(ctx, keys[0])
}

// List returns a list of nodes
func (n *NodeStore) List(ctx context.Context, filters ...routing.NodeInfoFilter) ([]models.NodeInfo, error) {
	keys, err := n.kv.Keys(ctx)
	if err != nil {
		// Return an empty list rather than an error if there are no keys in the bucket
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return []models.NodeInfo{}, nil
		}
		return nil, errors.Wrap(err, "failed to list node info from node store")
	}

	var errors *multierror.Error

	// Create a mega filter that combines all the filters into one
	megaFilter := func(info models.NodeInfo) bool {
		for _, filter := range filters {
			if !filter(info) {
				return false
			}
		}
		return true
	}

	nodes := make([]models.NodeInfo, 0, len(keys))
	for _, key := range keys {
		node, err := n.Get(ctx, key)
		if err != nil {
			errors = multierror.Append(errors, err)
		}

		if megaFilter(node) {
			nodes = append(nodes, node)
		}
	}

	return nodes, errors.ErrorOrNil()
}

// Delete deletes a node info from the repo.
func (n *NodeStore) Delete(ctx context.Context, nodeID string) error {
	if err := n.kv.Purge(ctx, nodeID); err != nil {
		return errors.Wrap(err, "failed to purge node info from node store")
	}

	return nil
}

var _ routing.NodeInfoStore = (*NodeStore)(nil)
