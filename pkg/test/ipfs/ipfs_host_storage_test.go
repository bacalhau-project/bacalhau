//go:build integration || !unit

package ipfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type IPFSHostStorageSuite struct {
	suite.Suite
	client *ipfs.Client
	Config types.BacalhauConfig
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestIPFSHostStorageSuite(t *testing.T) {
	suite.Run(t, new(IPFSHostStorageSuite))
}

// Before each test
func (suite *IPFSHostStorageSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	_, suite.Config = setup.SetupBacalhauRepoForTesting(suite.T())
	testutils.MustHaveIPFS(suite.T(), suite.Config.Node.IPFS.Connect)

	var err error
	suite.client, err = ipfs.NewClient(context.Background(), suite.Config.Node.IPFS.Connect)
	suite.Require().NoError(err)

}

type getStorageFunc func(ctx context.Context, api ipfs.Client) (
	storage.Storage, error)

func (suite *IPFSHostStorageSuite) TestIpfsApiCopyFile() {
	suite.runFileTest(
		func(ctx context.Context, api ipfs.Client) (
			storage.Storage, error) {

			return ipfs_storage.NewStorage(api, time.Duration(suite.Config.Node.VolumeSizeRequestTimeout))
		},
	)
}

func (suite *IPFSHostStorageSuite) TestIPFSAPICopyFolder() {
	suite.runFolderTest(
		func(ctx context.Context, api ipfs.Client) (
			storage.Storage, error) {

			return ipfs_storage.NewStorage(api, time.Duration(suite.Config.Node.VolumeSizeRequestTimeout))
		},
	)
}

func (suite *IPFSHostStorageSuite) runFileTest(getStorageDriver getStorageFunc) {
	ctx := context.Background()

	// add this file to the server
	EXAMPLE_TEXT := `hello world`
	fileCid, err := ipfs.AddTextToNodes(ctx, []byte(EXAMPLE_TEXT), *suite.client)
	suite.Require().NoError(err)

	// construct an ipfs docker storage client
	storageDriver, err := getStorageDriver(ctx, *suite.client)
	suite.Require().NoError(err)

	// the storage spec for the cid we added
	inputSource := models.InputSource{
		Source: &models.SpecConfig{
			Type:   models.StorageSourceIPFS,
			Params: ipfs_storage.Source{CID: fileCid}.ToMap(),
		},
		Target: "/data/file.txt",
	}

	suite.verifyHasCID(ctx, storageDriver, inputSource, fileCid)

	volume, err := storageDriver.PrepareStorage(ctx, suite.T().TempDir(), inputSource)
	suite.Require().NoError(err)

	// we should now be able to read our file content
	// from the file on the host via fuse
	r, err := os.ReadFile(volume.Source)
	suite.Require().NoError(err)
	suite.Require().Equal(string(r), EXAMPLE_TEXT)

	err = storageDriver.CleanupStorage(ctx, inputSource, volume)
	suite.Require().NoError(err)
}

func (suite *IPFSHostStorageSuite) runFolderTest(getStorageDriver getStorageFunc) {
	ctx := context.Background()

	dir := suite.T().TempDir()

	EXAMPLE_TEXT := `hello world`
	err := os.WriteFile(fmt.Sprintf("%s/file.txt", dir), []byte(EXAMPLE_TEXT), 0644)
	suite.Require().NoError(err)

	// add this file to the server
	folderCid, err := ipfs.AddFileToNodes(ctx, dir, *suite.client)
	suite.Require().NoError(err)

	// construct an ipfs docker storage client
	storageDriver, err := getStorageDriver(ctx, *suite.client)
	suite.Require().NoError(err)

	// the storage spec for the cid we added
	inputSource := models.InputSource{
		Source: &models.SpecConfig{
			Type:   models.StorageSourceIPFS,
			Params: ipfs_storage.Source{CID: folderCid}.ToMap(),
		},
		Target: "/data/folder",
	}

	suite.verifyHasCID(ctx, storageDriver, inputSource, folderCid)

	volume, err := storageDriver.PrepareStorage(ctx, suite.T().TempDir(), inputSource)
	suite.Require().NoError(err)

	// we should now be able to read our file content
	// from the file on the host via fuse

	r, err := os.ReadFile(filepath.Join(volume.Source, "file.txt"))
	suite.Require().NoError(err)
	suite.Require().Equal(string(r), EXAMPLE_TEXT)

	err = storageDriver.CleanupStorage(ctx, inputSource, volume)
	suite.Require().NoError(err)
}

func (suite *IPFSHostStorageSuite) verifyHasCID(ctx context.Context,
	storageDriver storage.Storage,
	inputSource models.InputSource,
	fileCid string) {
	// we check public dht for the cid providers, and can take time
	// for the cid to be discoverable
	var hasCid bool
	var err error
	timeoutAt := time.Now().Add(10 * time.Second)
	for !hasCid && time.Now().Before(timeoutAt) {
		hasCid, err = storageDriver.HasStorageLocally(ctx, inputSource)
		suite.Require().NoError(err)
		if !hasCid {
			time.Sleep(100 * time.Millisecond)
		}
	}
	suite.Require().Truef(hasCid, "cid %s not found in local storage", fileCid)
}
