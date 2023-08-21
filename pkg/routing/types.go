package routing

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/peer"
	libp2p_routing "github.com/libp2p/go-libp2p/core/routing"
)

type NodeInfoStore interface {
	libp2p_routing.PeerRouting
	// Add adds a node info to the repo.
	Add(ctx context.Context, nodeInfo models.NodeInfo) error
	// Get returns the node info for the given peer ID.
	Get(ctx context.Context, peerID peer.ID) (models.NodeInfo, error)
	// List returns a list of nodes
	List(ctx context.Context) ([]models.NodeInfo, error)
	// ListForEngine returns a list of nodes that support the given engine.
	ListForEngine(ctx context.Context, engine string) ([]models.NodeInfo, error)
	// Delete deletes a node info from the repo.
	Delete(ctx context.Context, peerID peer.ID) error
}
