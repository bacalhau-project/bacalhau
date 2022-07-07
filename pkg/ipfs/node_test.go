package ipfs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

const testString = "Hello World"

// TestFunctionality tests the in-process IPFS node/client as follows:
//   1. local IPFS can be created using the 'test' profile
//   2. files can be uploaded/downloaded from the IPFS network
func TestFunctionality(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	n1, err := NewLocalNode(cm, nil)
	require.NoError(t, err)

	addrs, err := n1.SwarmAddresses()
	require.NoError(t, err)

	var n2 *Node
	n2, err = NewLocalNode(cm, addrs) // connect to first node
	require.NoError(t, err)

	// Create a file in a temp dir to upload to the nodes:
	dirPath, err := os.MkdirTemp("", "ipfs-client-test")
	require.NoError(t, err)

	filePath := filepath.Join(dirPath, "test.txt")
	file, err := os.Create(filePath)
	require.NoError(t, err)
	defer file.Close()

	_, err = file.WriteString(testString)
	require.NoError(t, err)

	// Upload a file to the second client:
	cl2, err := n2.Client()
	require.NoError(t, err)
	require.NoError(t, cl2.WaitUntilAvailable(ctx))

	cid, err := cl2.Put(ctx, filePath)
	require.NoError(t, err)
	require.NotEmpty(t, cid)

	// Download the file from the first client:
	cl1, err := n1.Client()
	require.NoError(t, err)
	require.NoError(t, cl1.WaitUntilAvailable(ctx))

	outputPath := filepath.Join(dirPath, "output.txt")
	err = cl1.Get(ctx, cid, outputPath)
	require.NoError(t, err)

	// Check that the file was downloaded correctly:
	file, err = os.Open(outputPath)
	require.NoError(t, err)
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, testString, string(data))
}
