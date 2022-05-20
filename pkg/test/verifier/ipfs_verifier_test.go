package verifier

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/types"
	ipfs_verifier "github.com/filecoin-project/bacalhau/pkg/verifier/ipfs"
	"github.com/stretchr/testify/assert"
)

func TestIPFSVerifier(t *testing.T) {
	stack, cancelContext := SetupTest(
		t,
		1,
	)

	defer TeardownTest(stack, cancelContext)

	dir, err := ioutil.TempDir("", "bacalhau-ipfs-verifier-test")
	assert.NoError(t, err)

	err = os.WriteFile(dir+"/file.txt", []byte("hello world"), 0644)
	assert.NoError(t, err)

	verifier, err := ipfs_verifier.NewIPFSVerifier(cancelContext, stack.Nodes[0].IpfsNode.ApiAddress())
	assert.NoError(t, err)

	installed, err := verifier.IsInstalled()
	assert.NoError(t, err)
	assert.True(t, installed)

	result, err := verifier.ProcessResultsFolder(&types.Job{}, dir)
	assert.NoError(t, err)

	fmt.Printf("RESULT: %s\n", result)

}
