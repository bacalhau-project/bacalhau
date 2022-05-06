package test

import (
	"context"
	"fmt"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func runTest(t *testing.T, engine string, getStorageDriver func(ctx context.Context, api string) (storage.StorageProvider, error)) {
	EXAMPLE_TEXT := `hello world`
	// get a single IPFS server
	stack, cancelFunction := SetupTest_IPFS(
		t,
		1,
	)

	defer TeardownTest_IPFS(stack, cancelFunction)

	// add this file to the server
	fileCid, err := stack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	assert.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()

	storageDriver, err := getStorageDriver(stack.Ctx, ipfsNodeAddress)
	assert.NoError(t, err)

	// the storage spec for the cid we added
	storage := types.StorageSpec{
		Engine:    engine,
		Cid:       fileCid,
		MountPath: "/data/file.txt",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorage(storage)
	assert.NoError(t, err)
	assert.True(t, hasCid)

	// this should start a sidecar container with a fuse mount
	volume, err := storageDriver.PrepareStorage(storage)
	assert.NoError(t, err)

	// we should now be able to read our file content
	// from the file on the host via fuse
	result, err := system.RunCommandGetResults("sudo", []string{
		"cat",
		volume.Source,
	})
	assert.NoError(t, err)
	assert.Equal(t, result, EXAMPLE_TEXT)

	fmt.Printf("HERE IS RESULTS: %s\n", result)

	err = storageDriver.CleanupStorage(storage, volume)
	assert.NoError(t, err)
}

func TestIpfsFuseDocker(t *testing.T) {

	runTest(
		t,
		storage.IPFS_FUSE_DOCKER,
		func(ctx context.Context, api string) (storage.StorageProvider, error) {
			return fuse_docker.NewIpfsFuseDocker(ctx, api)
		},
	)

}

func TestIpfsApiCopy(t *testing.T) {

	runTest(
		t,
		storage.IPFS_API_COPY,
		func(ctx context.Context, api string) (storage.StorageProvider, error) {
			return api_copy.NewIpfsApiCopy(ctx, api)
		},
	)

}
