package discovery

import (
	"context"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

type StoreNodeDiscovererParams struct {
	Host         host.Host
	Store        requester.NodeInfoStore
	PeerStoreTTL time.Duration
}

type StoreNodeDiscoverer struct {
	host         host.Host
	store        requester.NodeInfoStore
	peerStoreTTL time.Duration
}

func NewStoreNodeDiscoverer(params StoreNodeDiscovererParams) *StoreNodeDiscoverer {
	return &StoreNodeDiscoverer{
		host:         params.Host,
		store:        params.Store,
		peerStoreTTL: params.PeerStoreTTL,
	}
}

// FindNodes returns the nodes that support the job's execution engine, and have enough TOTAL capacity to run the job.
func (d *StoreNodeDiscoverer) FindNodes(ctx context.Context, job model.Job) ([]peer.ID, error) {
	var peers []peer.ID
	nodeInfos, err := d.store.ListForEngine(ctx, job.Spec.Engine)
	if err != nil || len(nodeInfos) == 0 {
		return peers, err
	}

	jobResourceUsage := capacity.ParseResourceUsageConfig(job.Spec.Resources)
	for _, nodeInfo := range nodeInfos {
		if jobResourceUsage.LessThanEq(nodeInfo.ComputeNodeInfo.MaxJobRequirements) {
			// add peer info to the host's peerstore to be able to connect to it
			d.host.Peerstore().AddAddrs(nodeInfo.PeerInfo.ID, nodeInfo.PeerInfo.Addrs, d.peerStoreTTL)
			peers = append(peers, nodeInfo.PeerInfo.ID)
		}
	}

	return peers, nil
}

// compile time check that StoreNodeDiscoverer implements NodeDiscoverer
var _ requester.NodeDiscoverer = (*StoreNodeDiscoverer)(nil)
