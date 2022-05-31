package ipfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func runFileTest(t *testing.T, engine string, getStorageDriver func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error)) {
	EXAMPLE_TEXT := `hello world`
	// get a single IPFS server
	stack, cancelContext := SetupTest(
		t,
		1,
	)

	defer TeardownTest(stack, cancelContext)

	// add this file to the server
	fileCid, err := stack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	assert.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()

	storageDriver, err := getStorageDriver(stack.CancelContext, ipfsNodeAddress)
	assert.NoError(t, err)

	// the storage spec for the cid we added
	storage := types.StorageSpec{
		Engine: engine,
		Cid:    fileCid,
		Path:   "/data/file.txt",
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

func runFolderTest(t *testing.T, engine string, getStorageDriver func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error)) {
	EXAMPLE_TEXT := `hello world`

	dir, err := ioutil.TempDir("", "bacalhau-ipfs-test")
	assert.NoError(t, err)

	err = os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	assert.NoError(t, err)

	// get a single IPFS server
	stack, cancelContext := SetupTest(
		t,
		1,
	)

	defer TeardownTest(stack, cancelContext)

	// add this file to the server
	folderCid, err := stack.AddFolderToNodes(1, dir)
	assert.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()

	storageDriver, err := getStorageDriver(stack.CancelContext, ipfsNodeAddress)
	assert.NoError(t, err)

	// the storage spec for the cid we added
	storage := types.StorageSpec{
		Engine: engine,
		Cid:    folderCid,
		Path:   "/data/folder",
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
		fmt.Sprintf("%s/file.txt", volume.Source),
	})
	assert.NoError(t, err)
	assert.Equal(t, result, EXAMPLE_TEXT)

	fmt.Printf("HERE IS RESULTS: %s\n", result)

	err = storageDriver.CleanupStorage(storage, volume)
	assert.NoError(t, err)
}

// fuse tests disabled for now since we care about being able to run the tests
// on macOS, and aren't using the fuse driver anyway.

// func TestIpfsFuseDockerFile(t *testing.T) {

// 	runFileTest(
// 		t,
// 		storage.IPFS_FUSE_DOCKER,
// 		func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error) {
// 			return fuse_docker.NewIpfsFuseDocker(cancelContext, api)
// 		},
// 	)

// }

// func TestIpfsFuseDockerFolder(t *testing.T) {

// 	runFolderTest(
// 		t,
// 		storage.IPFS_FUSE_DOCKER,
// 		func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error) {
// 			return fuse_docker.NewIpfsFuseDocker(cancelContext, api)
// 		},
// 	)

// }

func TestIpfsApiCopyFile(t *testing.T) {

	runFileTest(
		t,
		storage.IPFS_API_COPY,
		func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error) {
			return api_copy.NewIpfsApiCopy(cancelContext, api)
		},
	)

}

func TestIpfsApiCopyFolder(t *testing.T) {

	runFolderTest(
		t,
		storage.IPFS_API_COPY,
		func(cancelContext *system.CancelContext, api string) (storage.StorageProvider, error) {
			return api_copy.NewIpfsApiCopy(cancelContext, api)
		},
	)
}
