//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type MaxUsageNodeRankerSuite struct {
	suite.Suite
	MaxUsageNodeRanker *MaxUsageNodeRanker
	smallPeer          models.NodeInfo
	medPeer            models.NodeInfo
	largePeer          models.NodeInfo
}

func (s *MaxUsageNodeRankerSuite) SetupSuite() {
	s.smallPeer = models.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("small")},
		ComputeNodeInfo: &models.ComputeNodeInfo{MaxJobRequirements: models.Resources{CPU: 1}},
	}
	s.medPeer = models.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("med")},
		ComputeNodeInfo: &models.ComputeNodeInfo{MaxJobRequirements: models.Resources{CPU: 2}},
	}
	s.largePeer = models.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("large")},
		ComputeNodeInfo: &models.ComputeNodeInfo{MaxJobRequirements: models.Resources{CPU: 3}},
	}
}

func (s *MaxUsageNodeRankerSuite) SetupTest() {
	s.MaxUsageNodeRanker = NewMaxUsageNodeRanker()
}

func TestMaxUsageNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(MaxUsageNodeRankerSuite))
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_VerySmallJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{CPU: "0.1"}
	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 10)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_SmallJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{CPU: "1"}
	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 10)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_MedJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{CPU: "2"}
	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_LargeJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{CPU: "3"}
	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", -1)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_VeryLargeJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{CPU: "3.1"}

	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", -1)
	assertEquals(s.T(), ranks, "large", -1)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodesUnknownJob() {
	job := mock.Job()
	job.Task().ResourcesConfig = &models.ResourcesConfig{}
	nodes := []models.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), *job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 0)
	assertEquals(s.T(), ranks, "med", 0)
	assertEquals(s.T(), ranks, "large", 0)
}
