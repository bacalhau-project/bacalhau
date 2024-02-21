package kvstore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/pkg/errors"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
)

type nodeInfoWrapper struct {
	Node     models.NodeInfo
	EvictAt  time.Time
	Revision uint64
}

type NodeStoreParams struct {
	TTL            time.Duration
	ConnectionInfo interface{}
}

type NodeStore struct {
	ttl time.Duration
	js  jetstream.JetStream
	kv  jetstream.KeyValue
}

func NewNodeStore(params NodeStoreParams) (*NodeStore, error) {
	url, ok := params.ConnectionInfo.(string)
	if !ok {
		return nil, errors.New("invalid connection info provided to KV Node Store")
	}

	// The connection we get from NATS is thread-safe (see https://pkg.go.dev/github.com/nats-io/nats.go#Conn)
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to connect to nats network at %s", url))
	}

	js, err := jetstream.New(nc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to jetstream")
	}

	kv, err := js.CreateKeyValue(context.Background(), jetstream.KeyValueConfig{
		Bucket: "nodes",
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create key-value store")
	}

	return &NodeStore{
		ttl: params.TTL,
		js:  js,
		kv:  kv,
	}, nil
}

func (n *NodeStore) FindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	return peer.AddrInfo{}, nil
}

// Add adds a node info to the repo.
func (n *NodeStore) Add(ctx context.Context, nodeInfo models.NodeInfo) error {
	wrapper := nodeInfoWrapper{
		Node:     nodeInfo,
		EvictAt:  time.Now().Add(n.ttl),
		Revision: 0,
	}

	data, err := json.Marshal(wrapper)
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

	var info nodeInfoWrapper
	err = json.Unmarshal(entry.Value(), &info)
	if err != nil {
		return models.NodeInfo{}, errors.Wrap(err, "failed to unmarshal node info from node store")
	}

	return info.Node, nil
}

// GetByPrefix returns the node info for the given node ID.
// Supports both full and short node IDs.
func (n *NodeStore) GetByPrefix(ctx context.Context, prefix string) (models.NodeInfo, error) {
	return models.NodeInfo{}, nil
}

// List returns a list of nodes
func (n *NodeStore) List(ctx context.Context) ([]models.NodeInfo, error) {
	keys, err := n.kv.Keys(ctx)
	if err != nil {
		// Return an empty list rather than an error if there are no keys in the bucket
		if errors.Is(err, jetstream.ErrNoKeysFound) {
			return []models.NodeInfo{}, nil
		}
		return nil, errors.Wrap(err, "failed to list node info from node store")
	}

	var errors *multierror.Error

	nodes := make([]models.NodeInfo, len(keys))
	for _, key := range keys {
		node, err := n.Get(ctx, key)
		if err != nil {
			errors = multierror.Append(errors, err)
		}

		nodes = append(nodes, node)
	}

	return nodes, errors.ErrorOrNil()
}

// Delete deletes a node info from the repo.
func (n *NodeStore) Delete(ctx context.Context, nodeID string) error {
	if err := n.kv.Delete(ctx, nodeID); err != nil {
		return errors.Wrap(err, "failed to delete node info from node store")
	}

	if err := n.kv.Purge(ctx, nodeID); err != nil {
		return errors.Wrap(err, "failed to purge node info from node store")
	}

	return nil
}

var _ routing.NodeInfoStore = (*NodeStore)(nil)
