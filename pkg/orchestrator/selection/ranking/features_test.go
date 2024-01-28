//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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
			NodeID:          "docker",
			ComputeNodeInfo: &models.ComputeNodeInfo{ExecutionEngines: []string{models.EngineDocker}},
		},
		{
			NodeID:          "wasm",
			ComputeNodeInfo: &models.ComputeNodeInfo{ExecutionEngines: []string{models.EngineWasm}},
		},
		{
			NodeID:          "ipfs",
			ComputeNodeInfo: &models.ComputeNodeInfo{StorageSources: []string{models.StorageSourceIPFS}},
		},
		{
			NodeID:          "url",
			ComputeNodeInfo: &models.ComputeNodeInfo{StorageSources: []string{models.StorageSourceURL}},
		},
		{
			NodeID: "combo",
			ComputeNodeInfo: &models.ComputeNodeInfo{
				ExecutionEngines: []string{models.EngineDocker, models.EngineWasm},
				Publishers:       []string{models.PublisherIPFS, models.PublisherS3},
				StorageSources:   []string{models.StorageSourceIPFS, models.StorageSourceURL},
			},
		},
		{
			NodeID: "unknown",
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
