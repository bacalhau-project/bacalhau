package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

type getStorageFunc func(cm *system.CleanupManager, api string) (
	storage.StorageProvider, error)

func runFileTest(t *testing.T, engine string, getStorageDriver getStorageFunc) {
	// get a single IPFS server
	stack, cm := SetupTest(t, 1)
	defer TeardownTest(stack, cm)

	// add this file to the server
	EXAMPLE_TEXT := `hello world`
	fileCid, err := stack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	assert.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()
	storageDriver, err := getStorageDriver(cm, ipfsNodeAddress)
	assert.NoError(t, err)

	// the storage spec for the cid we added
	storage := types.StorageSpec{
		Engine: engine,
		Cid:    fileCid,
		Path:   "/data/file.txt",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorage(context.TODO(), storage)
	assert.NoError(t, err)
	assert.True(t, hasCid)

	// this should start a sidecar container with a fuse mount
	volume, err := storageDriver.PrepareStorage(context.TODO(), storage)
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

	err = storageDriver.CleanupStorage(context.TODO(), storage, volume)
	assert.NoError(t, err)
}

func runFolderTest(t *testing.T, engine string,
	getStorageDriver getStorageFunc) {

	dir, err := ioutil.TempDir("", "bacalhau-ipfs-test")
	assert.NoError(t, err)

	EXAMPLE_TEXT := `hello world`
	err = os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	assert.NoError(t, err)

	// get a single IPFS server
	stack, cm := SetupTest(t, 1)
	defer TeardownTest(stack, cm)

	// add this file to the server
	folderCid, err := stack.AddFolderToNodes(1, dir)
	assert.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsNode.ApiAddress()
	storageDriver, err := getStorageDriver(cm, ipfsNodeAddress)
	assert.NoError(t, err)

	// the storage spec for the cid we added
	storage := types.StorageSpec{
		Engine: engine,
		Cid:    folderCid,
		Path:   "/data/folder",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorage(context.TODO(), storage)
	assert.NoError(t, err)
	assert.True(t, hasCid)

	// this should start a sidecar container with a fuse mount
	volume, err := storageDriver.PrepareStorage(context.TODO(), storage)
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

	err = storageDriver.CleanupStorage(context.TODO(), storage, volume)
	assert.NoError(t, err)
}

func TestIpfsFuseDockerFile(t *testing.T) {
	t.Skip("fuse tests disabled for now since we care about being able to run the tests on macOS, and aren't using the fuse driver anyway.")

	runFileTest(
		t,
		storage.IPFS_FUSE_DOCKER,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return fuse_docker.NewStorageProvider(cm, api)
		},
	)
}

func TestIpfsFuseDockerFolder(t *testing.T) {
	t.Skip("fuse tests disabled for now since we care about being able to run the tests on macOS, and aren't using the fuse driver anyway.")

	runFolderTest(
		t,
		storage.IPFS_FUSE_DOCKER,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return fuse_docker.NewStorageProvider(cm, api)
		},
	)

}

func TestIpfsApiCopyFile(t *testing.T) {

	runFileTest(
		t,
		storage.IPFS_API_COPY,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return api_copy.NewStorageProvider(cm, api)
		},
	)

}

func TestIpfsApiCopyFolder(t *testing.T) {

	runFolderTest(
		t,
		storage.IPFS_API_COPY,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return api_copy.NewStorageProvider(cm, api)
		},
	)
}
