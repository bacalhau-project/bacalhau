//go:build unit || !integration

package ranking

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

type MinVersionNodeRankerSuite struct {
	suite.Suite
	MinVersionNodeRanker *MinVersionNodeRanker
}

func (s *MinVersionNodeRankerSuite) SetupTest() {
	s.MinVersionNodeRanker = NewMinVersionNodeRanker(MinVersionNodeRankerParams{
		MinVersion: models.BuildVersionInfo{
			Major:      "1",
			Minor:      "3",
			GitVersion: "v1.3.12",
			GitCommit:  "2e444a7364d789d90a9b8600b09a9e9cb31afb09",
		},
	})
}

func TestMinVersionNodeRankerSuite(t *testing.T) {
	suite.Run(t, new(MinVersionNodeRankerSuite))
}

type minVersionNodeRankerTestCase struct {
	name     string
	expected int
	version  models.BuildVersionInfo
}

var minVersionNodeRankerTestCases = []minVersionNodeRankerTestCase{
	{
		name:     "oldMajor",
		version:  models.BuildVersionInfo{Major: "0", Minor: "9"},
		expected: -1,
	},
	{
		name:     "oldMinor",
		version:  models.BuildVersionInfo{Major: "1", Minor: "2"},
		expected: -1,
	},
	{
		name:     "oldGitVersion",
		version:  models.BuildVersionInfo{Major: "1", Minor: "3", GitVersion: "v1.3.11"},
		expected: -1,
	},
	{
		name:     "nilVersion",
		version:  models.BuildVersionInfo{},
		expected: 0,
	},
	{
		name:     "match",
		version:  models.BuildVersionInfo{Major: "1", Minor: "3", GitVersion: "v1.3.12"},
		expected: 10,
	},
	{
		name:     "newMajor",
		version:  models.BuildVersionInfo{Major: "4", Minor: "0"},
		expected: 10,
	},
	{
		name:     "newMinor",
		version:  models.BuildVersionInfo{Major: "1", Minor: "4"},
		expected: 10,
	},
	{
		name:     "newGitVersion",
		version:  models.BuildVersionInfo{Major: "1", Minor: "3", GitVersion: "v1.3.13"},
		expected: 10,
	},
	{
		name:     "developmentVersion",
		version:  developmentVersion,
		expected: 10,
	},
}

func (s *MinVersionNodeRankerSuite) TestRankNodes() {
	var nodes []models.NodeInfo
	for _, t := range minVersionNodeRankerTestCases {
		nodes = append(nodes, models.NodeInfo{
			PeerInfo: peer.AddrInfo{ID: peer.ID(t.name)},
			Version:  t.version,
		})
	}

	ranks, err := s.MinVersionNodeRanker.RankNodes(context.Background(), models.Job{}, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	for _, t := range minVersionNodeRankerTestCases {
		assertEquals(s.T(), ranks, t.name, t.expected)
	}
}

// if nil version is passed to the ranker, it should accept all nodes
func (s *MinVersionNodeRankerSuite) TestRankNodes_NilMinVersion() {
	s.MinVersionNodeRanker = NewMinVersionNodeRanker(MinVersionNodeRankerParams{
		MinVersion: models.BuildVersionInfo{},
	})
	var nodes []models.NodeInfo
	for _, t := range minVersionNodeRankerTestCases {
		nodes = append(nodes, models.NodeInfo{
			PeerInfo: peer.AddrInfo{ID: peer.ID(t.name)},
			Version:  t.version,
		})
	}

	ranks, err := s.MinVersionNodeRanker.RankNodes(context.Background(), models.Job{}, nodes)
	s.NoError(err)
	s.Equal(len(nodes), len(ranks))
	for _, t := range minVersionNodeRankerTestCases {
		expectedRank := 10
		if t.name == "nilVersion" {
			expectedRank = 0
		}
		assertEquals(s.T(), ranks, t.name, expectedRank)
	}
}
