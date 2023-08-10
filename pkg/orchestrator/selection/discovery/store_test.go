//go:build unit || !integration

package discovery

import (
	"context"
	"math"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// both nodes are returned when asked for docker nodes
	job := models.Job{}
	job.Tasks[0].Engine = &models.SpecConfig{Type: models.EngineDocker}
	peerIDs, err := s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1, nodeInfo2}, peerIDs)

	// only node2 is returned when asked for noop nodes
	job.Tasks[0].Engine = &models.SpecConfig{Type: models.EngineNoop}
	peerIDs, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Empty(peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_Empty() {
	peerIDs, err := s.discoverer.FindNodes(context.Background(), models.Job{})
	s.NoError(err)
	s.Empty(peerIDs)
}

func generateNodeInfo(id string, engines ...string) models.NodeInfo {
	return models.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID: peer.ID(id),
			Addrs: []multiaddr.Multiaddr{
				multiaddr.StringCast("/ip4/0.0.0.0/tcp/1234"),
			},
		},
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: &models.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
