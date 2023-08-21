//go:build unit || !integration

package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type InMemoryNodeInfoStoreSuite struct {
	suite.Suite
	store *NodeInfoStore
}

func (s *InMemoryNodeInfoStoreSuite) SetupTest() {
	s.store = NewNodeInfoStore(NodeInfoStoreParams{
		TTL: 1 * time.Hour,
	})
}

func TestInMemoryNodeInfoStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryNodeInfoStoreSuite))
}

func (s *InMemoryNodeInfoStoreSuite) Test_Get() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// test Get
	res1, err := s.store.Get(ctx, nodeInfo1.PeerInfo.ID)
	s.NoError(err)
	s.Equal(nodeInfo1, res1)

	res2, err := s.store.Get(ctx, nodeInfo2.PeerInfo.ID)
	s.NoError(err)
	s.Equal(nodeInfo2, res2)
}

func (s *InMemoryNodeInfoStoreSuite) Test_GetNotFound() {
	ctx := context.Background()
	_, err := s.store.Get(ctx, peer.ID("node1"))
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)

}

func (s *InMemoryNodeInfoStoreSuite) Test_List() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1, nodeInfo2}, allNodeInfos)
}

func (s *InMemoryNodeInfoStoreSuite) Test_ListForEngine() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", models.EngineWasm)
	nodeInfo3 := generateNodeInfo("node3", models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))
	s.NoError(s.store.Add(ctx, nodeInfo3))

	dockerNodes, err := s.store.ListForEngine(ctx, models.EngineDocker)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1, nodeInfo3}, dockerNodes)

	wasmNodes, err := s.store.ListForEngine(ctx, models.EngineWasm)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo2, nodeInfo3}, wasmNodes)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Delete() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	nodeInfo2 := generateNodeInfo("node2", models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo1))
	s.NoError(s.store.Add(ctx, nodeInfo2))

	// delete first node
	s.NoError(s.store.Delete(ctx, nodeInfo1.PeerInfo.ID))
	dockerNodes, err := s.store.ListForEngine(ctx, models.EngineDocker)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo2}, dockerNodes)

	wasmNodes, err := s.store.ListForEngine(ctx, models.EngineWasm)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo2}, wasmNodes)

	// delete second node
	s.NoError(s.store.Delete(ctx, nodeInfo2.PeerInfo.ID))
	dockerNodes, err = s.store.ListForEngine(ctx, models.EngineDocker)
	s.NoError(err)
	s.Empty(dockerNodes)

	wasmNodes, err = s.store.ListForEngine(ctx, models.EngineWasm)
	s.NoError(err)
	s.Empty(wasmNodes)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Replace() {
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo1))

	nodeInfo2 := generateNodeInfo("node1", models.EngineWasm)
	nodeInfo2.PeerInfo.ID = nodeInfo1.PeerInfo.ID
	s.NoError(s.store.Add(ctx, nodeInfo2))

	res, err := s.store.Get(ctx, nodeInfo1.PeerInfo.ID)
	s.NoError(err)
	s.Equal(nodeInfo2, res)

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo2}, allNodeInfos)

	// test ListForEngine
	dockerNodes, err := s.store.ListForEngine(ctx, models.EngineDocker)
	s.NoError(err)
	s.Empty(dockerNodes)

	wasmNodes, err := s.store.ListForEngine(ctx, models.EngineWasm)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo2}, wasmNodes)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Eviction() {
	ttl := 1 * time.Second
	s.store = NewNodeInfoStore(NodeInfoStoreParams{
		TTL: ttl,
	})
	ctx := context.Background()
	nodeInfo1 := generateNodeInfo("node1", models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// test Get
	res, err := s.store.Get(ctx, nodeInfo1.PeerInfo.ID)
	s.NoError(err)
	s.Equal(nodeInfo1, res)

	// wait for eviction
	time.Sleep(ttl + 100*time.Millisecond)
	_, err = s.store.Get(ctx, nodeInfo1.PeerInfo.ID)
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func generateNodeInfo(id string, engines ...string) models.NodeInfo {
	return models.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID: peer.ID(id),
		},
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: &models.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
