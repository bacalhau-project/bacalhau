//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type EnginesNodeRankerSuite struct {
	suite.Suite
	EnginesNodeRanker *EnginesNodeRanker
	dockerPeer        model.NodeInfo
	wasmPeer          model.NodeInfo
	comboPeer         model.NodeInfo
	unknownPeer       model.NodeInfo
}

func (s *EnginesNodeRankerSuite) SetupSuite() {
	s.dockerPeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("docker")},
		ComputeNodeInfo: model.ComputeNodeInfo{ExecutionEngines: []model.Engine{model.EngineDocker}},
	}
	s.wasmPeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("wasm")},
		ComputeNodeInfo: model.ComputeNodeInfo{ExecutionEngines: []model.Engine{model.EngineWasm}},
	}
	s.comboPeer = model.NodeInfo{
		PeerInfo:        peer.AddrInfo{ID: peer.ID("combo")},
		ComputeNodeInfo: model.ComputeNodeInfo{ExecutionEngines: []model.Engine{model.EngineDocker, model.EngineWasm}},
	}
	s.unknownPeer = model.NodeInfo{
		PeerInfo: peer.AddrInfo{ID: peer.ID("unknown")},
	}
}

func (s *EnginesNodeRankerSuite) SetupTest() {
	s.EnginesNodeRanker = NewEnginesNodeRanker()
}

func TestEnginesNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(EnginesNodeRankerSuite))
}

func (s *EnginesNodeRankerSuite) TestRankNodes_Docker() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineDocker}}
	nodes := []model.NodeInfo{s.dockerPeer, s.wasmPeer, s.comboPeer, s.unknownPeer}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "docker", 10)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
func (s *EnginesNodeRankerSuite) TestRankNodes_Wasm() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineWasm}}
	nodes := []model.NodeInfo{s.dockerPeer, s.wasmPeer, s.comboPeer, s.unknownPeer}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *EnginesNodeRankerSuite) TestRankNodes_Noop() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineNoop}}
	nodes := []model.NodeInfo{s.dockerPeer, s.wasmPeer, s.comboPeer, s.unknownPeer}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", -1)
	assertEquals(s.T(), ranks, "unknown", 0)
}
