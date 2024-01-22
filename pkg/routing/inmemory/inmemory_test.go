//go:build unit || !integration

package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/stretchr/testify/suite"
)

var nodeIDs = []string{
	"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}

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
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeInfo(s.T(), nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// test Get
	res1, err := s.store.Get(ctx, nodeInfo0.ID())
	s.NoError(err)
	s.Equal(nodeInfo0, res1)

	res2, err := s.store.Get(ctx, nodeInfo1.ID())
	s.NoError(err)
	s.Equal(nodeInfo1, res2)
}

func (s *InMemoryNodeInfoStoreSuite) Test_GetNotFound() {
	ctx := context.Background()
	_, err := s.store.Get(ctx, nodeIDs[0])
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)

}

func (s *InMemoryNodeInfoStoreSuite) Test_GetByPrefix_SingleMatch() {
	ctx := context.Background()
	nodeInfo := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo))

	res, err := s.store.GetByPrefix(ctx, "QmdZQ7")
	s.NoError(err)
	s.Equal(nodeInfo, res)
}

func (s *InMemoryNodeInfoStoreSuite) Test_GetByPrefix_MultipleMatches() {
	ctx := context.Background()
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeInfo(s.T(), nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	_, err := s.store.GetByPrefix(ctx, "Qm")
	s.Error(err)
	s.IsType(routing.ErrMultipleNodesFound{}, err)
}

func (s *InMemoryNodeInfoStoreSuite) Test_GetByPrefix_NoMatch() {
	ctx := context.Background()
	_, err := s.store.GetByPrefix(ctx, "nonexistent")
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func (s *InMemoryNodeInfoStoreSuite) Test_GetByPrefix_ExpiredNode() {
	ctx := context.Background()
	nodeInfo := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo))

	// simulate expiration by directly manipulating the store's data
	s.store.mu.Lock()
	infoWrapper := s.store.nodeInfoMap[nodeInfo.ID()]
	infoWrapper.evictAt = time.Now().Add(-time.Minute) // set eviction time in the past
	s.store.nodeInfoMap[nodeInfo.ID()] = infoWrapper
	s.store.mu.Unlock()

	_, err := s.store.GetByPrefix(ctx, "QmdZQ7")
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func (s *InMemoryNodeInfoStoreSuite) Test_List() {
	ctx := context.Background()
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeInfo(s.T(), nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo0, nodeInfo1}, allNodeInfos)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Delete() {
	ctx := context.Background()
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeInfo(s.T(), nodeIDs[1], models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo0))
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// delete first node
	s.NoError(s.store.Delete(ctx, nodeInfo0.ID()))
	nodes, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1}, nodes)

	// delete second node
	s.NoError(s.store.Delete(ctx, nodeInfo1.ID()))
	nodes, err = s.store.List(ctx)
	s.NoError(err)
	s.Empty(nodes)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Replace() {
	ctx := context.Background()
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo0))

	nodeInfo1 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineWasm)
	nodeInfo1.NodeID = nodeInfo0.NodeID
	s.NoError(s.store.Add(ctx, nodeInfo1))

	res, err := s.store.Get(ctx, nodeInfo0.ID())
	s.NoError(err)
	s.Equal(nodeInfo1, res)

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1}, allNodeInfos)
}

func (s *InMemoryNodeInfoStoreSuite) Test_Eviction() {
	ttl := 1 * time.Second
	s.store = NewNodeInfoStore(NodeInfoStoreParams{
		TTL: ttl,
	})
	ctx := context.Background()
	nodeInfo0 := generateNodeInfo(s.T(), nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo0))

	// test Get
	res, err := s.store.Get(ctx, nodeInfo0.ID())
	s.NoError(err)
	s.Equal(nodeInfo0, res)

	// wait for eviction
	time.Sleep(ttl + 100*time.Millisecond)
	_, err = s.store.Get(ctx, nodeInfo0.ID())
	s.Error(err)
	s.IsType(routing.ErrNodeNotFound{}, err)
}

func generateNodeInfo(t *testing.T, peerID string, engines ...string) models.NodeInfo {
	return models.NodeInfo{
		NodeID:   peerID,
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: &models.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
