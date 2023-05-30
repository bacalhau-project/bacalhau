//go:build unit || !integration

package filecoinunsealed

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

var ctx context.Context
var tempDir string
var driver *StorageProvider
var cm *system.CleanupManager

type FilecoinUnsealedSuite struct {
	suite.Suite
}

func (suite *FilecoinUnsealedSuite) prepareCid(c cid.Cid) spec.Storage {
	folderPath := filepath.Join(tempDir, c.String())
	err := os.MkdirAll(folderPath, os.ModePerm)
	require.NoError(suite.T(), err)
	// TODO determine if we are keeping model.StorageSourceFilecoinUnsealed or using the IPFS storage spec instead
	out, err := (&ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", folderPath)
	require.NoError(suite.T(), err)
	return out
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestFilecoinUnsealedSuite(t *testing.T) {
	suite.Run(t, new(FilecoinUnsealedSuite))
}

// Before each test
func (suite *FilecoinUnsealedSuite) SetupTest() {
	var setupErr error
	logger.ConfigureTestLogging(suite.T())
	cm = system.NewCleanupManager()
	ctx = context.Background()
	tempDir = suite.T().TempDir()
	driver, setupErr = NewStorage(cm, filepath.Join(tempDir, "{{.CID}}"))
	require.NoError(suite.T(), setupErr)
}

func (suite *FilecoinUnsealedSuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
}

func (suite *FilecoinUnsealedSuite) TestHasStorageLocally() {
	storage := suite.prepareCid(testutil.TestCID1)
	hasStorageTrue, err := driver.HasStorageLocally(ctx, storage)
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorageTrue, "file that exists should return true for HasStorageLocally")
	storage = suite.prepareCid(testutil.TestCID2)
	hasStorageFalse, err := driver.HasStorageLocally(ctx, storage)
	require.NoError(suite.T(), err)
	require.False(suite.T(), hasStorageFalse, "file that does not exist should return false for HasStorageLocally")
}

func (suite *FilecoinUnsealedSuite) TestGetVolumeSize() {
	// NB: the CID here doesn't correspond to the contents of the file, but is used to validate this functionality that doesn't go to network.
	fileContents := "hello world"
	storage := suite.prepareCid(testutil.TestCID1)
	filePath := filepath.Join(storage.Mount, "file")
	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	require.NoError(suite.T(), err)
	volumeSize, err := driver.GetVolumeSize(ctx, storage)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(len(fileContents)), volumeSize, "the volume size should be the size of the file")
}

func (suite *FilecoinUnsealedSuite) TestPrepareStorage() {
	spec := suite.prepareCid(testutil.TestCID1)
	volume, err := driver.PrepareStorage(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), spec.Mount, volume.Source, "the volume source should be the same as the spec path")
}
