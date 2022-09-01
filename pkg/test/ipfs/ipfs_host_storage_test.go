package ipfs

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IPFSHostStorageSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestIPFSHostStorageSuite(t *testing.T) {
	suite.Run(t, new(IPFSHostStorageSuite))
}

// Before all suite
func (suite *IPFSHostStorageSuite) SetupAllSuite() {

}

// Before each test
func (suite *IPFSHostStorageSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *IPFSHostStorageSuite) TearDownTest() {
}

func (suite *IPFSHostStorageSuite) TearDownAllSuite() {

}

type getStorageFunc func(cm *system.CleanupManager, ctx context.Context, api string) (
	storage.StorageProvider, error)

func (suite *IPFSHostStorageSuite) TestIpfsApiCopyFile() {
	runFileTest(
		suite.T(),
		model.StorageSourceIPFS,
		func(cm *system.CleanupManager, ctx context.Context, api string) (
			storage.StorageProvider, error) {

			return apicopy.NewStorageProvider(cm, ctx, api)
		},
	)

}

func (suite *IPFSHostStorageSuite) TestIPFSAPICopyFolder() {
	runFolderTest(
		suite.T(),
		model.StorageSourceIPFS,
		func(cm *system.CleanupManager, ctx context.Context, api string) (
			storage.StorageProvider, error) {

			return apicopy.NewStorageProvider(cm, ctx, api)
		},
	)
}

func runFileTest(t *testing.T, engine model.StorageSourceType, getStorageDriver getStorageFunc) {
	ctx := context.Background()
	// get a single IPFS server
	stack, cm := SetupTest(t, ctx, 1)
	defer TeardownTest(stack, cm)

	tr := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, tr, "pkg/test/ipfs/runFolderTest")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	// add this file to the server
	EXAMPLE_TEXT := `hello world`
	fileCid, err := stack.AddTextToNodes(ctx, 1, []byte(EXAMPLE_TEXT))
	require.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsClient.APIAddress()
	storageDriver, err := getStorageDriver(cm, ctx, ipfsNodeAddress)
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := model.StorageSpec{
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

	err = storageDriver.CleanupStorage(ctx, storage, volume)
	require.NoError(t, err)
}

func runFolderTest(t *testing.T, engine model.StorageSourceType, getStorageDriver getStorageFunc) {
	ctx := context.Background()
	// get a single IPFS server
	stack, cm := SetupTest(t, ctx, 1)
	defer TeardownTest(stack, cm)

	tr := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(ctx, tr, "pkg/test/ipfs/runFolderTest")
	defer rootSpan.End()
	cm.RegisterCallback(system.CleanupTraceProvider)

	dir, err := ioutil.TempDir("", "bacalhau-ipfs-test")
	require.NoError(t, err)

	EXAMPLE_TEXT := `hello world`
	err = os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	require.NoError(t, err)

	// add this file to the server
	folderCid, err := stack.AddFolderToNodes(ctx, 1, dir)
	require.NoError(t, err)

	// construct an ipfs docker storage client
	ipfsNodeAddress := stack.Nodes[0].IpfsClient.APIAddress()
	storageDriver, err := getStorageDriver(cm, ctx, ipfsNodeAddress)
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := model.StorageSpec{
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
