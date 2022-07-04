package ipfs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

const testString = "Hello World"

func TestClient(t *testing.T) {
	ctx := context.Background()
	cm := system.NewCleanupManager()
	defer cm.Cleanup()

	cl1, err := NewClient(cm, Config{
		BootstrapNodes: make([]string, 0), // first node has no peers
	})
	assert.NoError(t, err)

	var cl2 *Client
	cl2, err = NewClient(cm, Config{
		BootstrapNodes: []string{cl1.Multiaddr()}, // connect to first node
	})
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
}
