//go:build unit || !integration

package discovery

import (
	"context"
	"math"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/routing/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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
	job := mock.Job()
	job.Task().Engine.Type = models.EngineDocker

	peerIDs, err := s.discoverer.FindNodes(context.Background(), *job)
	s.NoError(err)
	s.ElementsMatch([]models.NodeInfo{nodeInfo1, nodeInfo2}, peerIDs)

	// only node2 is returned when asked for noop nodes
	job2 := mock.Job()
	peerIDs, err = s.discoverer.FindNodes(context.Background(), *job2)
	s.NoError(err)
	s.Empty(peerIDs)
}

func (s *StoreNodeDiscovererSuite) TestFindNodes_Empty() {
	peerIDs, err := s.discoverer.FindNodes(context.Background(), *mock.Job())
	s.NoError(err)
	s.Empty(peerIDs)
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
