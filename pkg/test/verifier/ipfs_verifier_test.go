package verifier

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/stretchr/testify/assert"
)

func TestIPFSVerifier(t *testing.T) {
	ctx := context.Background()
	stack, cm := SetupTest(t, 1)
	defer TeardownTest(stack, cm)

	inputDir, err := ioutil.TempDir("", "bacalhau-ipfs-verifier-test")
	assert.NoError(t, err)

	outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-verifier-test")
	assert.NoError(t, err)

	fixtureContent := "hello world"
	err = os.WriteFile(inputDir+"/file.txt", []byte(fixtureContent), 0644)
	assert.NoError(t, err)

	verifier, err := ipfs.NewVerifier(
		cm, stack.Nodes[0].IpfsNode.ApiAddress())
	assert.NoError(t, err)

	installed, err := verifier.IsInstalled(ctx)
	assert.NoError(t, err)
	assert.True(t, installed)

	resultHash, err := verifier.ProcessResultsFolder(ctx,
		"fake-job-id", inputDir)
	assert.NoError(t, err)

	err = verifier.IPFSClient.DownloadTar(ctx, outputDir, resultHash)
	assert.NoError(t, err)

	outputContent, err := os.ReadFile(outputDir + "/" + resultHash + "/file.txt")
	assert.NoError(t, err)

	assert.Equal(t, fixtureContent, string(outputContent))
}
