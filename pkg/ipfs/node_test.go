//go:build unit || !integration

package ipfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/stretchr/testify/suite"
)

const testString = "Hello World"

type NodeSuite struct {
	suite.Suite
}

func (s *NodeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	s.Require().NoError(system.InitConfigForTesting(s.T()))
}

// TestFunctionality tests the in-process IPFS node/client as follows:
//  1. local IPFS can be created using the 'test' profile
//  2. files can be uploaded/downloaded from the IPFS network
//  3. a local IPFS doesn't auto-discover any peers
func (s *NodeSuite) TestFunctionality() {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	cm := system.NewCleanupManager()
	s.T().Cleanup(func() {
		cm.Cleanup(context.Background())
	})

	n1, err := NewLocalNode(ctx, cm, nil)
	s.Require().NoError(err)

	addrs, err := n1.SwarmAddresses()
	s.Require().NoError(err)

	n2, err := NewLocalNode(ctx, cm, addrs) // connect to first node
	s.Require().NoError(err)

	n3, err := NewLocalNode(ctx, cm, nil) // to test that it doesn't auto-discover anyone
	s.Require().NoError(err)

	// Create a file in a temp dir to upload to the nodes:
	dirPath := s.T().TempDir()

	filePath := filepath.Join(dirPath, "test.txt")

	s.Require().NoError(os.WriteFile(filePath, []byte(testString), 0644))

	// Upload a file to the second client:
	cl2 := n2.Client()

	cid, err := cl2.Put(ctx, filePath)
	s.Require().NoError(err)
	s.Require().NotEmpty(cid)

	// Validate file was uploaded and pinned
	_, isPinned, err := cl2.API.Pin().IsPinned(ctx, icorepath.New(cid))
	s.Require().NoError(err)
	s.Require().True(isPinned)

	// Download the file from the first client:
	cl1 := n1.Client()

	outputPath := filepath.Join(dirPath, "output.txt")
	err = cl1.Get(ctx, cid, outputPath)
	s.Require().NoError(err)

	// Check that the file was downloaded correctly:
	data, err := os.ReadFile(outputPath)
	s.Require().NoError(err)
	s.Require().Equal(testString, string(data))

	s.Never(func() bool {
		peers, err := n2.Client().API.Swarm().Peers(ctx)
		s.Require().NoError(err)

		return !s.Len(peers, 1)
	}, 500*time.Millisecond, 10*time.Millisecond, "a local node should only connect to the passed in peers")

	s.Never(func() bool {
		peers, err := n3.Client().API.Swarm().Peers(ctx)
		s.Require().NoError(err)

		return !s.Empty(peers)
	}, 500*time.Millisecond, 10*time.Millisecond, "a local node should never auto-discover anyone")
}

// a normal test function and pass our suite to suite.Run
func TestNodeSuite(t *testing.T) {
	suite.Run(t, new(NodeSuite))
}
