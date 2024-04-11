//go:build unit || !integration

package discovery

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type StoreNodeDiscovererSuite struct {
	suite.Suite
	discoverer *StoreNodeDiscoverer
	store      *inmemory.NodeStore
}

func (s *StoreNodeDiscovererSuite) SetupTest() {
	s.store = inmemory.NewNodeStore(inmemory.NodeStoreParams{
		TTL: math.MaxInt64,
	})
	s.discoverer = NewStoreNodeDiscoverer(StoreNodeDiscovererParams{
		Store: s.store,
	})
}

func TestStoreNodeDiscovererSuite(t *testing.T) {
	suite.Run(t, new(StoreNodeDiscovererSuite))
}

func (s *StoreNodeDiscovererSuite) TestListNodes() {
	ctx := context.Background()
	nodeInfo1 := generateNodeState("node1", models.EngineDocker)
	s.NoError(s.store.Add(ctx, nodeInfo1))

	// both nodes are returned when asked for docker nodes
	job := mock.Job()
	job.Task().Engine.Type = models.EngineDocker

	peerIDs, err := s.discoverer.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo1}, peerIDs)

	nodeInfo2 := generateNodeState("node2", models.EngineDocker, models.EngineWasm)
	s.NoError(s.store.Add(ctx, nodeInfo2))
	// only node2 is returned when asked for noop nodes
	peerIDs, err = s.discoverer.ListNodes(context.Background())
	s.NoError(err)
	s.ElementsMatch([]models.NodeState{nodeInfo1, nodeInfo2}, peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestListNodes_Empty() {
	peerIDs, err := s.discoverer.ListNodes(context.Background())
	s.NoError(err)
	s.Empty(peerIDs)
}

func generateNodeState(id string, engines ...string) models.NodeState {
	return models.NodeState{
		Info: generateNodeInfo(id, engines...),
	}
}

func generateNodeInfo(id string, engines ...string) models.NodeInfo {
	return models.NodeInfo{
		NodeID:   id,
		NodeType: models.NodeTypeCompute,
		ComputeNodeInfo: &models.ComputeNodeInfo{
			ExecutionEngines: engines,
		},
	}
}
