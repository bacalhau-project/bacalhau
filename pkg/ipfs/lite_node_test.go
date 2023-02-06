//go:build unit || !integration

package ipfs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/suite"
)

const testContent = "Hello from IPFS Gateway Checker"
const testCid = "bafybeifx7yeb55armcsxwwitkymga5xf53dxiarykms3ygqic223w5sk3m"

type LiteNodeSuite struct {
	suite.Suite
}

func (s *LiteNodeSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	s.Require().NoError(system.InitConfigForTesting(s.T()))
}

func (s *LiteNodeSuite) TestFunctionality() {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	n1, err := NewLiteNode(ctx, LiteNodeParams{})
	s.Require().NoError(err)

	// Create a file in a temp dir to download content to
	dirPath := s.T().TempDir()
	filePath := filepath.Join(dirPath, "test.txt")

	n1Client := n1.Client()
	s.Require().NoError(n1Client.Get(ctx, testCid, filePath))

	// Check that the file was downloaded correctly:
	data, err := os.ReadFile(filePath)
	s.Require().NoError(err)
	s.Require().Equal(testContent, strings.TrimSpace(string(data)))

	// Check that a second node can be created withtin the same process with no errors
	n1Addrs, err := n1.SwarmAddresses()
	s.Require().NoError(err)
	s.Require().NotEmpty(n1Addrs)
	n2, err := NewLiteNode(ctx, LiteNodeParams{
		PeerAddrs: n1Addrs[0:1], // only 1 address is enough
	})
	s.Require().NoError(err)

	// check the node is connected to the first one
	// it might take some time for the node 1 info to show up in the peer store of node 2
	var peer1Info peer.AddrInfo
	waitUntil := time.Now().Add(2 * time.Second)
	for time.Now().Before(waitUntil) {
		peer1Info = n2.ipfsNode.Peerstore.PeerInfo(n1.ipfsNode.Identity)
		if len(peer1Info.Addrs) > 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.Require().NotEmpty(peer1Info.Addrs)

	// Check that the second node can download the same content
	filePath2 := filepath.Join(dirPath, "test2.txt")
	s.Require().NoError(n2.Client().Get(ctx, testCid, filePath2))
	data2, err := os.ReadFile(filePath)
	s.Require().NoError(err)
	s.Require().Equal(data, data2)

	// close the nodes
	s.Require().NoError(n1.Close())
	s.Require().NoError(n2.Close())
}

// a normal test function and pass our suite to suite.Run
func TestLiteNodeSuite(t *testing.T) {
	suite.Run(t, new(LiteNodeSuite))
}
