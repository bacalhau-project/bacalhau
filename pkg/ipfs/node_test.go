package ipfs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	addrs, err := n1.SwarmAddresses()
	assert.NoError(t, err)

	var n2 *Node
	n2, err = NewLocalNode(cm, addrs) // connect to first node
	assert.NoError(t, err)

	// Create a file in a temp dir to upload to the nodes:
	dirPath, err := os.MkdirTemp("", "ipfs-client-test")
	assert.NoError(t, err)

	filePath := filepath.Join(dirPath, "test.txt")
	file, err := os.Create(filePath)
	assert.NoError(t, err)
	defer file.Close()

	_, err = file.WriteString(testString)
	assert.NoError(t, err)

	// Upload a file to the second client:
	cl2, err := n2.Client()
	assert.NoError(t, err)
	assert.NoError(t, cl2.WaitUntilAvailable(ctx))

	cid, err := cl2.Put(ctx, filePath)
	assert.NoError(t, err)
	assert.NotEmpty(t, cid)

	// Download the file from the first client:
	cl1, err := n1.Client()
	assert.NoError(t, err)
	assert.NoError(t, cl1.WaitUntilAvailable(ctx))

	outputPath := filepath.Join(dirPath, "output.txt")
	err = cl1.Get(ctx, cid, outputPath)
	assert.NoError(t, err)

	// Check that the file was downloaded correctly:
	file, err = os.Open(outputPath)
	assert.NoError(t, err)
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	assert.NoError(t, err)
	assert.Equal(t, testString, string(data))
}
