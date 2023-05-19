//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type FeatureNodeRankerSuite struct {
	suite.Suite
	EnginesNodeRanker   *featureNodeRanker[model.Engine]
	VerifiersNodeRanker *featureNodeRanker[model.Verifier]
	PublisherNodeRanker *featureNodeRanker[model.Publisher]
	StorageNodeRanker   *featureNodeRanker[model.StorageSourceType]
}

func (s *FeatureNodeRankerSuite) Nodes() []model.NodeInfo {
	return []model.NodeInfo{
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("docker")},
			ComputeNodeInfo: &model.ComputeNodeInfo{ExecutionEngines: []model.Engine{model.EngineDocker}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("wasm")},
			ComputeNodeInfo: &model.ComputeNodeInfo{ExecutionEngines: []model.Engine{model.EngineWasm}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("ipfs")},
			ComputeNodeInfo: &model.ComputeNodeInfo{StorageSources: []model.StorageSourceType{model.StorageSourceIPFS}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("url")},
			ComputeNodeInfo: &model.ComputeNodeInfo{StorageSources: []model.StorageSourceType{model.StorageSourceURLDownload}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("deterministic")},
			ComputeNodeInfo: &model.ComputeNodeInfo{Verifiers: []model.Verifier{model.VerifierDeterministic}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("estuary")},
			ComputeNodeInfo: &model.ComputeNodeInfo{Publishers: []model.Publisher{model.PublisherEstuary}},
		},
		{
			PeerInfo: peer.AddrInfo{ID: peer.ID("combo")},
			ComputeNodeInfo: &model.ComputeNodeInfo{
				ExecutionEngines: []model.Engine{model.EngineDocker, model.EngineWasm},
				Verifiers:        []model.Verifier{model.VerifierNoop, model.VerifierDeterministic},
				Publishers:       []model.Publisher{model.PublisherIpfs, model.PublisherEstuary},
				StorageSources:   []model.StorageSourceType{model.StorageSourceIPFS, model.StorageSourceURLDownload},
			},
		},
		{
			PeerInfo: peer.AddrInfo{ID: peer.ID("unknown")},
		},
	}
}

func (s *FeatureNodeRankerSuite) SetupSuite() {
	s.EnginesNodeRanker = NewEnginesNodeRanker()
	s.StorageNodeRanker = NewStoragesNodeRanker()
	s.VerifiersNodeRanker = NewVerifiersNodeRanker()
	s.PublisherNodeRanker = NewPublishersNodeRanker()
}

func TestEnginesNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(FeatureNodeRankerSuite))
}

func (s *FeatureNodeRankerSuite) TestEngineDocker() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineDocker}}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", 10)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
func (s *FeatureNodeRankerSuite) TestEngineWasm() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineWasm}}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestEngineNoop() {
	job := model.Job{Spec: model.Spec{Engine: model.EngineNoop}}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", -1)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestVerifierDeterministic() {
	job := model.Job{Spec: model.Spec{Verifier: model.VerifierDeterministic}}
	ranks, err := s.VerifiersNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "deterministic", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestPublisherEstuary() {
	job := model.Job{Spec: model.Spec{PublisherSpec: model.PublisherSpec{Type: model.PublisherEstuary}}}
	ranks, err := s.PublisherNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "estuary", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestStorageIPFS() {
	job := model.Job{Spec: model.Spec{Inputs: []model.StorageSpec{
		{StorageSource: model.StorageSourceIPFS},
	}}}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", 10)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestStorageIPFSAndURL() {
	job := model.Job{Spec: model.Spec{Inputs: []model.StorageSpec{
		{StorageSource: model.StorageSourceIPFS},
		{StorageSource: model.StorageSourceURLDownload},
	}}}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", -1)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
