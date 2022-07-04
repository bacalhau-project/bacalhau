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

// TestClient tests some basic functionality of the in-process IPFS client:
//   1. local IPFS can be created using the 'test' profile
//   2. files can be uploaded/downloaded from the IPFS network
//   3. uploading to a local IPFS network doesn't pollute the public one
func TestClient(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()

	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	cl1, err := NewLocalClient(cm, nil)
	assert.NoError(t, err)

	addrs, err := cl1.P2pAddrs()
	assert.NoError(t, err)

	var cl2 *Client
	cl2, err = NewLocalClient(cm, addrs)
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
	cid, err := cl2.Put(ctx, filePath)
	assert.NoError(t, err)
	assert.NotEmpty(t, cid)

	// Download the file from the first client:
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

	// Create a new client on the public IPFS network:
	cl3, err := NewClient(cm)
	assert.NoError(t, err)

	// Check that the file didn't pollute the global IPFS network:
	assert.Error(t, cl3.Get(ctx, cid, outputPath))
}
