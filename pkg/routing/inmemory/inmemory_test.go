//go:build unit || !integration

package inmemory_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
)

var nodeIDs = []string{
	"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}

type InMemoryNodeStoreSuite struct {
	suite.Suite
	store *inmemory.NodeStore
}

func (s *InMemoryNodeStoreSuite) SetupTest() {
	s.store = inmemory.NewNodeStore(inmemory.NodeStoreParams{
		TTL: 1 * time.Hour,
	})
}

func TestInMemoryNodeStoreSuite(t *testing.T) {
	suite.Run(t, new(InMemoryNodeStoreSuite))
}

func (s *InMemoryNodeStoreSuite) Test_Get() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// test Get
	res1, err := s.store.Get(ctx, nodeInfo0.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo0, res1)

	res2, err := s.store.Get(ctx, nodeInfo1.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo1, res2)
}

func (s *InMemoryNodeStoreSuite) Test_GetNotFound() {
	ctx := context.Background()
	_, err := s.store.Get(ctx, nodeIDs[0])
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)

}

func (s *InMemoryNodeStoreSuite) Test_GetByPrefix_SingleMatch() {
	ctx := context.Background()
	nodeInfo := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo))

	res, err := s.store.GetByPrefix(ctx, "QmdZQ7")
	s.NoError(err)
	s.Equal(nodeInfo, res)
}

func (s *InMemoryNodeStoreSuite) Test_GetByPrefix_MultipleMatches() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	_, err := s.store.GetByPrefix(ctx, "Qm")
	s.Error(err)
	s.IsType(routing.ErrMultipleNodesFound{}, err)
}

func (s *InMemoryNodeStoreSuite) Test_GetByPrefix_NoMatch() {
	ctx := context.Background()
	_, err := s.store.GetByPrefix(ctx, "nonexistent")
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func (s *InMemoryNodeStoreSuite) Test_GetByPrefix_ExpiredNode() {
	ctx := context.Background()
	store := inmemory.NewNodeStore(inmemory.NodeStoreParams{
		TTL: 10 * time.Millisecond,
	})

	nodeInfo := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(store.Add(ctx, nodeInfo))

	// Wait for the item to expire
	time.Sleep(20 * time.Millisecond)

	_, err := store.GetByPrefix(ctx, "QmdZQ7")
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func (s *InMemoryNodeStoreSuite) Test_List() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo0, nodeInfo1}, allNodeInfos)
}

func (s *InMemoryNodeStoreSuite) Test_ListWithFilters() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// Match one record
	filterPartialID := func(node models.NodeState) bool {
		return strings.HasPrefix(node.Info.ID(), string(nodeIDs[0][0:8]))
	}
	nodes, err := s.store.List(ctx, filterPartialID)
	s.NoError(err)
	s.Equal(1, len(nodes))
	s.Equal(nodeIDs[0], nodes[0].Info.ID())

	// Match all records
	filterPartialID = func(node models.NodeState) bool {
		return strings.HasPrefix(node.Info.ID(), "Qm")
	}
	nodes, err = s.store.List(ctx, filterPartialID)
	s.NoError(err)
	s.Equal(2, len(nodes))

	// Match no records
	filterPartialID = func(node models.NodeState) bool {
		return strings.HasPrefix(node.Info.ID(), "XYZ")
	}
	nodes, err = s.store.List(ctx, filterPartialID)
	s.NoError(err)
	s.Equal(0, len(nodes))
}

func (s *InMemoryNodeStoreSuite) Test_Delete() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// delete first node
	s.NoError(s.store.Delete(ctx, nodeInfo0.Info.ID()))
	nodes, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo1}, nodes)

	// delete second node
	s.NoError(s.store.Delete(ctx, nodeInfo1.Info.ID()))
	nodes, err = s.store.List(ctx)
	s.NoError(err)
	s.Empty(nodes)
}

func (s *InMemoryNodeStoreSuite) Test_Replace() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo0))

	nodeInfo1 := generateNodeState(nodeIDs[0], models.EngineWasm)
	nodeInfo1.Info.NodeID = nodeInfo0.Info.NodeID
	s.NoError(s.store.Add(ctx, nodeInfo1))

	res, err := s.store.Get(ctx, nodeInfo0.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo1, res)

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo1}, allNodeInfos)
}

func (s *InMemoryNodeStoreSuite) Test_Eviction() {
	ttl := 1 * time.Second
	s.store = inmemory.NewNodeStore(inmemory.NodeStoreParams{
		TTL: ttl,
	})
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo0))

	// test Get
	res, err := s.store.Get(ctx, nodeInfo0.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo0, res)

	// wait for eviction
	time.Sleep(ttl + 100*time.Millisecond)
	_, err = s.store.Get(ctx, nodeInfo0.Info.ID())
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func generateNodeState(peerID string, engines ...string) models.NodeState {
	return models.NodeState{
		Info: generateNodeInfo(peerID, engines...),
	}
}

func generateNodeInfo(peerID string, engines ...string) models.NodeInfo {
	return models.NodeInfo{
		NodeID:   peerID,
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: &models.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
