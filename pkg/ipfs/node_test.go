package ipfs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	icorepath "github.com/ipfs/interface-go-ipfs-core/path"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

const testString = "Hello World"

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type NodeSuite struct {
	suite.Suite
}

// Before all suite
func (suite *NodeSuite) SetupAllSuite() {

}

// Before each test
func (suite *NodeSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
}

func (suite *NodeSuite) TearDownTest() {
}

func (suite *NodeSuite) TearDownAllSuite() {

}

// TestFunctionality tests the in-process IPFS node/client as follows:
//  1. local IPFS can be created using the 'test' profile
//  2. files can be uploaded/downloaded from the IPFS network
func (suite *NodeSuite) TestFunctionality() {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	n1, err := NewLocalNode(ctx, cm, nil)
	require.NoError(suite.T(), err)

	addrs, err := n1.SwarmAddresses()
	require.NoError(suite.T(), err)

	var n2 *Node
	n2, err = NewLocalNode(ctx, cm, addrs) // connect to first node
	require.NoError(suite.T(), err)

	// Create a file in a temp dir to upload to the nodes:
	dirPath := suite.T().TempDir()

	filePath := filepath.Join(dirPath, "test.txt")
	file, err := os.Create(filePath)
	require.NoError(suite.T(), err)
	defer file.Close()

	_, err = file.WriteString(testString)
	require.NoError(suite.T(), err)

	// Upload a file to the second client:
	cl2, err := n2.Client()
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), cl2.WaitUntilAvailable(ctx))

	cid, err := cl2.Put(ctx, filePath)
	require.NoError(suite.T(), err)
	require.NotEmpty(suite.T(), cid)

	// Validate file was uploaded and pinned
	_, isPinned, err := cl2.API.Pin().IsPinned(ctx, icorepath.New(cid))
	require.NoError(suite.T(), err)
	require.True(suite.T(), isPinned)

	// Download the file from the first client:
	cl1, err := n1.Client()
	require.NoError(suite.T(), err)
	require.NoError(suite.T(), cl1.WaitUntilAvailable(ctx))

	outputPath := filepath.Join(dirPath, "output.txt")
	err = cl1.Get(ctx, cid, outputPath)
	require.NoError(suite.T(), err)

	// Check that the file was downloaded correctly:
	file, err = os.Open(outputPath)
	require.NoError(suite.T(), err)
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), testString, string(data))
}

// a normal test function and pass our suite to suite.Run
func TestNodeSuite(t *testing.T) {
	suite.Run(t, new(NodeSuite))
}
