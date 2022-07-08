package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/apicopy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fusedocker"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type getStorageFunc func(cm *system.CleanupManager, api string) (
	storage.StorageProvider, error)

func TestIpfsApiCopyFile(t *testing.T) {

	runFileTest(
		t,
		storage.IPFSAPICopy,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return apicopy.NewStorageProvider(cm, api)
		},
	)

}

func TestIpfsApiCopyFolder(t *testing.T) {

	runFolderTest(
		t,
		storage.IPFSAPICopy,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return apicopy.NewStorageProvider(cm, api)
		},
	)
}

func TestIpfsFuseDockerFile(t *testing.T) {
	t.Skip("fuse tests disabled for now since we care about being able to run the tests on macOS, and aren't using the fuse driver anyway.")

	runFileTest(
		t,
		storage.IPFSFuseDocker,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return fusedocker.NewStorageProvider(cm, api)
		},
	)
}

func TestIpfsFuseDockerFolder(t *testing.T) {
	t.Skip("fuse tests disabled for now since we care about being able to run the tests on macOS, and aren't using the fuse driver anyway.")

	runFolderTest(
		t,
		storage.IPFSFuseDocker,
		func(cm *system.CleanupManager, api string) (
			storage.StorageProvider, error) {

			return fusedocker.NewStorageProvider(cm, api)
		},
	)

}

func runFileTest(t *testing.T, engine string, getStorageDriver getStorageFunc) {
	// get a single IPFS server
	ctx, span := newSpan(engine)
	defer span.End()

	stack, cm := SetupTest(t, 1)
	defer TeardownTest(stack, cm)

	// add this file to the server
	EXAMPLE_TEXT := `hello world`
	fileCid, err := stack.AddTextToNodes(1, []byte(EXAMPLE_TEXT))
	require.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsClient.APIAddress()
	storageDriver, err := getStorageDriver(cm, ipfsNodeAddress)
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := storage.StorageSpec{
		Engine: engine,
		Cid:    fileCid,
		Path:   "/data/file.txt",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorageLocally(ctx, storage)
	require.NoError(t, err)
	require.True(t, hasCid)

	// this should start a sidecar container with a fuse mount
	volume, err := storageDriver.PrepareStorage(ctx, storage)
	require.NoError(t, err)

	// we should now be able to read our file content
	// from the file on the host via fuse
	result, err := system.RunCommandGetResults("sudo", []string{
		"cat",
		volume.Source,
	})
	require.NoError(t, err)
	require.Equal(t, result, EXAMPLE_TEXT)

	fmt.Printf("HERE IS RESULTS: %s\n", result)

	err = storageDriver.CleanupStorage(ctx, storage, volume)
	require.NoError(t, err)
}

func runFolderTest(t *testing.T, engine string, getStorageDriver getStorageFunc) {
	ctx, span := newSpan(engine)
	defer span.End()

	dir, err := ioutil.TempDir("", "bacalhau-ipfs-test")
	require.NoError(t, err)

	EXAMPLE_TEXT := `hello world`
	err = os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	require.NoError(t, err)

	// get a single IPFS server
	stack, cm := SetupTest(t, 1)
	defer TeardownTest(stack, cm)

	// add this file to the server
	folderCid, err := stack.AddFolderToNodes(1, dir)
	require.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsClient.APIAddress()
	storageDriver, err := getStorageDriver(cm, ipfsNodeAddress)
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := storage.StorageSpec{
		Engine: engine,
		Cid:    folderCid,
		Path:   "/data/folder",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorageLocally(ctx, storage)
	require.NoError(t, err)
	require.True(t, hasCid)

	// this should start a sidecar container with a fuse mount
	volume, err := storageDriver.PrepareStorage(ctx, storage)
	require.NoError(t, err)

	// we should now be able to read our file content
	// from the file on the host via fuse
	result, err := system.RunCommandGetResults("sudo", []string{
		"cat",
		fmt.Sprintf("%s/file.txt", volume.Source),
	})
	require.NoError(t, err)
	require.Equal(t, result, EXAMPLE_TEXT)

	fmt.Printf("HERE IS RESULTS: %s\n", result)

	err = storageDriver.CleanupStorage(ctx, storage, volume)
	require.NoError(t, err)
}

func newSpan(name string) (context.Context, trace.Span) {
	return system.Span(context.Background(), "ipfs_host_storage_test", name)
}
