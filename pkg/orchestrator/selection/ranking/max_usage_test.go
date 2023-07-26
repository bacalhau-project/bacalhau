//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type MaxUsageNodeRankerSuite struct {
	suite.Suite
	MaxUsageNodeRanker *MaxUsageNodeRanker
	smallPeer          model.NodeInfo
	medPeer            model.NodeInfo
	largePeer          model.NodeInfo
}

func (s *MaxUsageNodeRankerSuite) SetupSuite() {
	s.smallPeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("small")},
		ComputeNodeInfo: &model.ComputeNodeInfo{MaxJobRequirements: model.ResourceUsageData{CPU: 1}},
	}
	s.medPeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("med")},
		ComputeNodeInfo: &model.ComputeNodeInfo{MaxJobRequirements: model.ResourceUsageData{CPU: 2}},
	}
	s.largePeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("large")},
		ComputeNodeInfo: &model.ComputeNodeInfo{MaxJobRequirements: model.ResourceUsageData{CPU: 3}},
	}
}

func (s *MaxUsageNodeRankerSuite) SetupTest() {
	s.MaxUsageNodeRanker = NewMaxUsageNodeRanker()
}

func TestMaxUsageNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(MaxUsageNodeRankerSuite))
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_VerySmallJob() {
	job := model.Job{Spec: model.Spec{Resources: model.ResourceUsageConfig{CPU: "0.1"}}}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 10)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_SmallJob() {
	job := model.Job{Spec: model.Spec{Resources: model.ResourceUsageConfig{CPU: "1"}}}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 10)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_MedJob() {
	job := model.Job{Spec: model.Spec{Resources: model.ResourceUsageConfig{CPU: "2"}}}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", 10)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_LargeJob() {
	job := model.Job{Spec: model.Spec{Resources: model.ResourceUsageConfig{CPU: "3"}}}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", -1)
	assertEquals(s.T(), ranks, "large", 10)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodes_VeryLargeJob() {
	job := model.Job{Spec: model.Spec{Resources: model.ResourceUsageConfig{CPU: "3.1"}}}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", -1)
	assertEquals(s.T(), ranks, "med", -1)
	assertEquals(s.T(), ranks, "large", -1)
}

func (s *MaxUsageNodeRankerSuite) TestRankNodesUnknownJob() {
	job := model.Job{}
	nodes := []model.NodeInfo{s.smallPeer, s.medPeer, s.largePeer}
	ranks, err := s.MaxUsageNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "small", 0)
	assertEquals(s.T(), ranks, "med", 0)
	assertEquals(s.T(), ranks, "large", 0)
}
