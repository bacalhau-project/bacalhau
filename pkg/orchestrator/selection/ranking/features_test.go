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

type FeatureNodeRankerSuite struct {
	suite.Suite
	EnginesNodeRanker   *featureNodeRanker
	PublisherNodeRanker *featureNodeRanker
	StorageNodeRanker   *featureNodeRanker
}

func (s *FeatureNodeRankerSuite) Nodes() []models.NodeInfo {
	return []models.NodeInfo{
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("docker")},
			ComputeNodeInfo: &models.ComputeNodeInfo{ExecutionEngines: []string{models.EngineDocker}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("wasm")},
			ComputeNodeInfo: &models.ComputeNodeInfo{ExecutionEngines: []string{models.EngineWasm}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("ipfs")},
			ComputeNodeInfo: &models.ComputeNodeInfo{StorageSources: []string{models.StorageSourceIPFS}},
		},
		{
			PeerInfo:        peer.AddrInfo{ID: peer.ID("url")},
			ComputeNodeInfo: &models.ComputeNodeInfo{StorageSources: []string{models.StorageSourceURL}},
		},
		{
			PeerInfo: peer.AddrInfo{ID: peer.ID("combo")},
			ComputeNodeInfo: &models.ComputeNodeInfo{
				ExecutionEngines: []string{models.EngineDocker, models.EngineWasm},
				Publishers:       []string{models.PublisherIPFS, models.PublisherS3},
				StorageSources:   []string{models.StorageSourceIPFS, models.StorageSourceURL},
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
	s.PublisherNodeRanker = NewPublishersNodeRanker()
}

func TestEnginesNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(FeatureNodeRankerSuite))
}

func (s *FeatureNodeRankerSuite) TestEngineDocker() {
	job := mock.Job()
	job.Task().Engine.Type = models.EngineDocker
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), *job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", 10)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
func (s *FeatureNodeRankerSuite) TestEngineWasm() {
	job := mock.Job()
	job.Task().Engine.Type = models.EngineWasm
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), *job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", 10)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestEngineNoop() {
	job := mock.Job()
	job.Task().Engine.Type = models.EngineNoop
	ranks, err := s.EnginesNodeRanker.RankNodes(context.Background(), *job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "docker", -1)
	assertEquals(s.T(), ranks, "wasm", -1)
	assertEquals(s.T(), ranks, "combo", -1)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestStorageIPFS() {
	job := mock.Job()
	job.Task().InputSources = []*models.InputSource{
		{Source: &models.SpecConfig{Type: models.StorageSourceIPFS}},
	}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), *job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", 10)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}

func (s *FeatureNodeRankerSuite) TestStorageIPFSAndURL() {
	job := mock.Job()
	job.Task().InputSources = []*models.InputSource{
		{Source: &models.SpecConfig{Type: models.StorageSourceIPFS}},
		{Source: &models.SpecConfig{Type: models.StorageSourceURL}},
	}
	ranks, err := s.StorageNodeRanker.RankNodes(context.Background(), *job, s.Nodes())
	s.NoError(err)
	s.Equal(len(s.Nodes()), len(ranks))
	assertEquals(s.T(), ranks, "ipfs", -1)
	assertEquals(s.T(), ranks, "url", -1)
	assertEquals(s.T(), ranks, "combo", 10)
	assertEquals(s.T(), ranks, "unknown", 0)
}
