//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/docker"
	enginetesting "github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/engine/wasm"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/testing"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"
)

type FeatureNodeRankerSuite struct {
	suite.Suite
	EnginesNodeRanker   *featureNodeRanker[cid.Cid]
	VerifiersNodeRanker *featureNodeRanker[model.Verifier]
	PublisherNodeRanker *featureNodeRanker[model.Publisher]
	StorageNodeRanker   *featureNodeRanker[cid.Cid]
}

func (s *FeatureNodeRankerSuite) Nodes() []model.NodeInfo {
	return []model.NodeInfo{
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("docker")},
			ComputeNodeInfo: &model.ComputeNodeInfo{ExecutionEngines: []cid.Cid{docker.EngineType}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("wasm")},
			ComputeNodeInfo: &model.ComputeNodeInfo{ExecutionEngines: []cid.Cid{wasm.EngineType}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("ipfs")},
			ComputeNodeInfo: &model.ComputeNodeInfo{StorageSources: []cid.Cid{ipfs.StorageType}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("url")},
			ComputeNodeInfo: &model.ComputeNodeInfo{StorageSources: []cid.Cid{url.StorageType}},
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
				ExecutionEngines: []cid.Cid{docker.EngineType, wasm.EngineType},
				Verifiers:        []model.Verifier{model.VerifierNoop, model.VerifierDeterministic},
				Publishers:       []model.Publisher{model.PublisherIpfs, model.PublisherEstuary},
				StorageSources:   []cid.Cid{ipfs.StorageType, url.StorageType},
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
	job := model.Job{Spec: model.Spec{Engine: enginetesting.DockerMakeEngine(s.T())}}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", 10)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
func (s *FeatureNodeRankerSuite) TestEngineWasm() {
	job := model.Job{Spec: model.Spec{Engine: enginetesting.WasmMakeEngine(s.T(),
		enginetesting.WasmWithEntrypoint("_start"),
	)}}
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestEngineNoop() {
	job := model.Job{Spec: model.Spec{Engine: enginetesting.NoopMakeEngine(s.T(), "noop")}}
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
	ipfsspec, err := (&ipfs.IPFSStorageSpec{CID: storagetesting.TestCID1}).AsSpec("TODO", "TODO")
	s.Require().NoError(err)
	job := model.Job{Spec: model.Spec{Inputs: []spec.Storage{ipfsspec}}}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", 10)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestStorageIPFSAndURL() {
	ipfsspec, err := (&ipfs.IPFSStorageSpec{CID: storagetesting.TestCID1}).AsSpec("TODO", "TODO")
	s.Require().NoError(err)
	urlspec, err := (&url.URLStorageSpec{URL: "https://example.com"}).AsSpec("TODO", "TODO")
	s.Require().NoError(err)
	job := model.Job{Spec: model.Spec{Inputs: []spec.Storage{
		ipfsspec,
		urlspec,
	}}}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", -1)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
