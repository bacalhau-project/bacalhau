//go:build unit || !integration

package kvstore_test

import (
	"context"
	"strings"
	"testing"

	"github.com/nats-io/nats-server/v2/server"
	natsserver "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/nodes/kvstore"
)

const TEST_PORT = 8369

var nodeIDs = []string{
	"QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
	"QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
	"QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
}

type KVNodeInfoStoreSuite struct {
	suite.Suite
	nats  *server.Server
	store nodes.Store
}

func (s *KVNodeInfoStoreSuite) SetupTest() {
	opts := &natsserver.DefaultTestOptions
	opts.Port = TEST_PORT
	opts.JetStream = true
	opts.StoreDir = s.T().TempDir()

	s.nats = natsserver.RunServer(opts)
	natsClient, err := nats.Connect(s.nats.Addr().String())
	s.Require().NoError(err)

	s.store, _ = kvstore.NewNodeStore(context.Background(), kvstore.NodeStoreParams{
		BucketName: "test_nodes",
		Client:     natsClient,
	})
}

func (s *KVNodeInfoStoreSuite) TearDownTest() {
	s.nats.Shutdown()
}

func TestKVNodeInfoStoreSuite(t *testing.T) {
	suite.Run(t, new(KVNodeInfoStoreSuite))
}

func (s *KVNodeInfoStoreSuite) Test_Get() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))
	s.NoError(s.store.Put(ctx, nodeInfo1))

	// test Get
	res1, err := s.store.Get(ctx, nodeInfo0.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo0.Info.ID(), res1.Info.ID())

	res2, err := s.store.Get(ctx, nodeInfo1.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo1.Info.ID(), res2.Info.ID())
}

func (s *KVNodeInfoStoreSuite) Test_GetNotFound() {
	ctx := context.Background()
	_, err := s.store.Get(ctx, nodeIDs[0])
	s.Error(err)
	s.IsType(nodes.ErrNodeNotFound{}, err)
}

func (s *KVNodeInfoStoreSuite) Test_GetByPrefix_SingleMatch() {
	ctx := context.Background()
	nodeInfo := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Put(ctx, nodeInfo))

	res, err := s.store.GetByPrefix(ctx, "QmdZQ7")
	s.NoError(err)
	s.Equal(nodeInfo, res)
}

func (s *KVNodeInfoStoreSuite) Test_GetByPrefix_MultipleMatches() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))
	s.NoError(s.store.Put(ctx, nodeInfo1))

	_, err := s.store.GetByPrefix(ctx, "Qm")
	s.Error(err)
	s.IsType(nodes.ErrMultipleNodesFound{}, err)
}

func (s *KVNodeInfoStoreSuite) Test_GetByPrefix_NoMatch_Empty() {
	ctx := context.Background()
	_, err := s.store.GetByPrefix(ctx, "nonexistent")
	s.Error(err)
	s.IsType(nodes.ErrNodeNotFound{}, err)
}

func (s *KVNodeInfoStoreSuite) Test_GetByPrefix_NoMatch_NotEmpty() {
	ctx := context.Background()

	nodeInfo0 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))

	_, err := s.store.GetByPrefix(ctx, "nonexistent")
	s.Error(err)
	s.IsType(nodes.ErrNodeNotFound{}, err)
}

func (s *KVNodeInfoStoreSuite) Test_List() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))
	s.NoError(s.store.Put(ctx, nodeInfo1))

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo0, nodeInfo1}, allNodeInfos)
}

func (s *KVNodeInfoStoreSuite) Test_ListWithFilters() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))
	s.NoError(s.store.Put(ctx, nodeInfo1))

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

func (s *KVNodeInfoStoreSuite) Test_Delete() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	nodeInfo1 := generateNodeState(nodeIDs[1], models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Put(ctx, nodeInfo0))
	s.NoError(s.store.Put(ctx, nodeInfo1))

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

func (s *KVNodeInfoStoreSuite) Test_Replace() {
	ctx := context.Background()
	nodeInfo0 := generateNodeState(nodeIDs[0], models.EngineDocker)
	s.NoError(s.store.Put(ctx, nodeInfo0))

	nodeInfo1 := generateNodeState(nodeIDs[0], models.EngineWasm)
	nodeInfo1.Info.NodeID = nodeInfo0.Info.NodeID
	s.NoError(s.store.Put(ctx, nodeInfo1))

	res, err := s.store.Get(ctx, nodeInfo0.Info.ID())
	s.NoError(err)
	s.Equal(nodeInfo1, res)

	// test List
	allNodeInfos, err := s.store.List(ctx)
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo1}, allNodeInfos)
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
