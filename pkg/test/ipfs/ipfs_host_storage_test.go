//go:build unit || !integration

package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	ipfs_storage "github.com/filecoin-project/bacalhau/pkg/storage/ipfs"
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

// Before each test
func (suite *IPFSHostStorageSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)
}

type getStorageFunc func(ctx context.Context, cm *system.CleanupManager, api ipfs.Client) (
	storage.Storage, error)

func (suite *IPFSHostStorageSuite) TestIpfsApiCopyFile() {
	runFileTest(
		suite.T(),
		model.StorageSourceIPFS,
		func(ctx context.Context, cm *system.CleanupManager, api ipfs.Client) (
			storage.Storage, error) {

			return ipfs_storage.NewStorage(cm, api)
		},
	)
}

func (suite *IPFSHostStorageSuite) TestIPFSAPICopyFolder() {
	runFolderTest(
		suite.T(),
		model.StorageSourceIPFS,
		func(ctx context.Context, cm *system.CleanupManager, api ipfs.Client) (
			storage.Storage, error) {

			return ipfs_storage.NewStorage(cm, api)
		},
	)
}

func runFileTest(t *testing.T, engine model.StorageSourceType, getStorageDriver getStorageFunc) {
	ctx := context.Background()
	// get a single IPFS server
	stack, cm := SetupTest(ctx, t, 1)
	defer TeardownTest(cm)

	// add this file to the server
	EXAMPLE_TEXT := `hello world`
	fileCid, err := ipfs.AddTextToNodes(ctx, []byte(EXAMPLE_TEXT), stack.IPFSClients[0])
	require.NoError(t, err)

	// construct an ipfs docker storage client
	storageDriver, err := getStorageDriver(ctx, cm, stack.IPFSClients[0])
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := model.StorageSpec{
		StorageSource: engine,
		CID:           fileCid,
		Path:          "/data/file.txt",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorageLocally(ctx, storage)
	require.NoError(t, err)
	require.True(t, hasCid)

	volume, err := storageDriver.PrepareStorage(ctx, storage)
	require.NoError(t, err)

	// we should now be able to read our file content
	// from the file on the host via fuse
	r, err := os.ReadFile(volume.Source)
	require.NoError(t, err)
	require.Equal(t, string(r), EXAMPLE_TEXT)

	err = storageDriver.CleanupStorage(ctx, storage, volume)
	require.NoError(t, err)
}

func runFolderTest(t *testing.T, engine model.StorageSourceType, getStorageDriver getStorageFunc) {
	ctx := context.Background()
	// get a single IPFS server
	stack, cm := SetupTest(ctx, t, 1)
	defer TeardownTest(cm)

	dir := t.TempDir()

	EXAMPLE_TEXT := `hello world`
	err := os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	require.NoError(t, err)

	// add this file to the server
	folderCid, err := ipfs.AddFileToNodes(ctx, dir, stack.IPFSClients[0])
	require.NoError(t, err)

	// construct an ipfs docker storage client
	storageDriver, err := getStorageDriver(ctx, cm, stack.IPFSClients[0])
	require.NoError(t, err)

	// the storage spec for the cid we added
	storage := model.StorageSpec{
		StorageSource: engine,
		CID:           folderCid,
		Path:          "/data/folder",
	}

	// does the storage client think we have the cid locally?
	hasCid, err := storageDriver.HasStorageLocally(ctx, storage)
	require.NoError(t, err)
	require.True(t, hasCid)

	volume, err := storageDriver.PrepareStorage(ctx, storage)
	require.NoError(t, err)

	// we should now be able to read our file content
	// from the file on the host via fuse

	r, err := os.ReadFile(filepath.Join(volume.Source, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, string(r), EXAMPLE_TEXT)

	err = storageDriver.CleanupStorage(ctx, storage, volume)
	require.NoError(t, err)
}
