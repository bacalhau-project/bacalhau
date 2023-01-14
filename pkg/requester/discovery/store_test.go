package discovery

import (
	"context"
	"math"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/requester/nodestore"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/suite"
)

type StoreNodeDiscovererSuite struct {
	suite.Suite
	discoverer *StoreNodeDiscoverer
	store      *nodestore.InMemoryNodeInfoStore
	host       host.Host
}

func (s *StoreNodeDiscovererSuite) SetupSuite() {
	ctx := context.Background()
	var err error
	s.host, err = libp2p.NewHostForTest(ctx)
	s.NoError(err)
}

func (s *StoreNodeDiscovererSuite) SetupTest() {
	s.store = nodestore.NewInMemoryNodeInfoStore(nodestore.InMemoryNodeInfoStoreParams{
		TTL: math.MaxInt64,
	})
	s.discoverer = NewStoreNodeDiscoverer(StoreNodeDiscovererParams{
		Host:         s.host,
		Store:        s.store,
		PeerStoreTTL: math.MaxInt64,
	})
}

func (s *StoreNodeDiscovererSuite) TearDownSuite() {
	s.NoError(s.host.Close())
}

func TestStoreNodeDiscovererSuite(t *testing.T) {
	suite.Run(t, new(StoreNodeDiscovererSuite))
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_ByEngine() {
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
	s.ElementsMatch([]peer.ID{nodeInfo1.PeerInfo.ID, nodeInfo2.PeerInfo.ID}, peerIDs)

	// no nodes are returned when asked for

	// only node2 is returned when asked for noop nodes
	job.Spec.Engine = model.EngineNoop
	peerIDs, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Empty(peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_ByCapacity() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", model.EngineDocker)
	nodeInfo1.ComputeNodeInfo.MaxJobRequirements.CPU = 1
	nodeInfo2 := generateNodeInfo("node2", model.EngineDocker)
	nodeInfo2.ComputeNodeInfo.MaxJobRequirements.CPU = 2
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// both nodes are returned when asked for small CPU requirements
	job := model.Job{}
	job.Spec.Engine = model.EngineDocker
	job.Spec.Resources.CPU = "0.1"
	peerIDs, err := s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.ElementsMatch([]peer.ID{nodeInfo1.PeerInfo.ID, nodeInfo2.PeerInfo.ID}, peerIDs)

	// only node2 is returned when asked for large CPU requirements
	job.Spec.Resources.CPU = "1.1"
	peerIDs, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.ElementsMatch([]peer.ID{nodeInfo2.PeerInfo.ID}, peerIDs)

	// no nodes are returned when asked for larger CPU requirements
	job.Spec.Resources.CPU = "2.1"
	peerIDs, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Empty(peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_UpdatePeerStore() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("UpdatePeerStore_node1", model.EngineDocker)
	nodeInfo2 := generateNodeInfo("UpdatePeerStore_node2", model.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// validate nodes don't exist in peerstore
	s.Empty(s.host.Peerstore().Addrs(nodeInfo1.PeerInfo.ID))
	s.Empty(s.host.Peerstore().Addrs(nodeInfo2.PeerInfo.ID))

	// node1 is added to the peerstore when asked for docker nodes
	job := model.Job{}
	job.Spec.Engine = model.EngineDocker
	_, err := s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Equal(nodeInfo1.PeerInfo.Addrs, s.host.Peerstore().Addrs(nodeInfo1.PeerInfo.ID))
	s.Empty(s.host.Peerstore().Addrs(nodeInfo2.PeerInfo.ID))

	// node2 is added to the peerstore when asked for wasm nodes
	job.Spec.Engine = model.EngineWasm
	_, err = s.discoverer.FindNodes(context.Background(), job)
	s.NoError(err)
	s.Equal(nodeInfo1.PeerInfo.Addrs, s.host.Peerstore().Addrs(nodeInfo1.PeerInfo.ID))
	s.Equal(nodeInfo2.PeerInfo.Addrs, s.host.Peerstore().Addrs(nodeInfo2.PeerInfo.ID))
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
		ComputeNodeInfo: model.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
