package test

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage/dockeripfs"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestIpfsSidecar(t *testing.T) {

	stack, cancelFunction := SetupTest_IPFS(
		t,
		1,
	)

	defer TeardownTest_IPFS(stack, cancelFunction)

	fileCid, err := stack.AddTextToNodes(1, []byte(`hello world`))
	assert.NoError(t, err)

	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()

	dockerStorage, err := dockeripfs.NewStorageDockerIPFS(stack.Ctx, ipfsNodeAddress)
	assert.NoError(t, err)

	storage := types.StorageSpec{
		Engine: "ipfs",
		Cid:    fileCid,
	}

	hasCid, err := dockerStorage.HasStorage(storage)
	assert.NoError(t, err)
	assert.True(t, hasCid)

	volume, err := dockerStorage.PrepareStorage(storage)
	assert.NoError(t, err)

	spew.Dump(volume)
}
