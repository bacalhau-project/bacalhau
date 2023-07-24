//go:build unit || !integration

package discovery

import (
	"context"
	"math"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/suite"
)

type StoreNodeDiscovererSuite struct {
	suite.Suite
	discoverer *StoreNodeDiscoverer
	store      *inmemory.NodeInfoStore
}

func (s *StoreNodeDiscovererSuite) SetupTest() {
	s.store = inmemory.NewNodeInfoStore(inmemory.NodeInfoStoreParams{
		TTL: math.MaxInt64,
	})
	s.discoverer = NewStoreNodeDiscoverer(StoreNodeDiscovererParams{
		Store: s.store,
	})
}

func TestStoreNodeDiscovererSuite(t *testing.T) {
	suite.Run(t, new(StoreNodeDiscovererSuite))
}

func (s *StoreNodeDiscovererSuite) TestFindNodes() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", model.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", model.EngineDocker, model.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// both nodes are returned when asked for docker nodes
	job := model.Job{}
	job.Spec.Engine = model.EngineDocker
	peerIDs, err := s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.ElementsMatch([]model.NodeInfo{nodeInfo1, nodeInfo2}, peerIDs)

	// only node2 is returned when asked for noop nodes
	job.Spec.Engine = model.EngineNoop
	peerIDs, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Empty(peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_Empty() {
	peerIDs, err := s.discoverer.FindNodes(context.Background(), model.Job{})
	s.NoError(err)
	s.Empty(peerIDs)
}

func generateNodeInfo(id string, engines ...model.Engine) model.NodeInfo {
	return model.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID: peer.ID(id),
			Addrs: []multiaddr.Multiaddr{
				multiaddr.StringCast("/ip4/0.0.0.0/tcp/1234"),
			},
		},
		NodeType: model.NodeTypeCompute,
		ComputeNodeInfo: &model.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
