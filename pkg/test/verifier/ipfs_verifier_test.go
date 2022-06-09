package verifier

import (
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
	ipfs_verifier "github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/stretchr/testify/assert"
)

func TestIPFSVerifier(t *testing.T) {
	stack, ctx, cancel := SetupTest(t, 1)
	defer TeardownTest(stack, cancel)

	inputDir, err := ioutil.TempDir("", "bacalhau-ipfs-verifier-test")
	assert.NoError(t, err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-verifier-test")
	assert.NoError(t, err)

	fixtureContent := "hello world"
	err = os.WriteFile(inputDir+"/file.txt", []byte(fixtureContent), 0644)
	assert.NoError(t, err)

	verifier, err := ipfs_verifier.NewIPFSVerifier(
		ctx, stack.Nodes[0].IpfsNode.ApiAddress())
	assert.NoError(t, err)

	installed, err := verifier.IsInstalled()
	assert.NoError(t, err)
	assert.True(t, installed)

	resultHash, err := verifier.ProcessResultsFolder(&types.Job{}, inputDir)
	assert.NoError(t, err)

	err = verifier.IPFSClient.DownloadTar(outputDir, resultHash)
	assert.NoError(t, err)

	outputContent, err := os.ReadFile(outputDir + "/" + resultHash + "/file.txt")
	assert.NoError(t, err)

	assert.Equal(t, fixtureContent, string(outputContent))
}
