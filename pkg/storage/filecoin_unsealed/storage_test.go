package filecoinunsealed

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var ctx context.Context
var tempDir string
var driver *StorageProvider
var cm *system.CleanupManager

type FilecoinUnsealedSuite struct {
	suite.Suite
}

func (suite *FilecoinUnsealedSuite) prepareCid(cid string) model.StorageSpec {
	folderPath := filepath.Join(tempDir, cid)
	err := os.MkdirAll(folderPath, os.ModePerm)
	require.NoError(suite.T(), err)
	return model.StorageSpec{
		StorageSource: model.StorageSourceFilecoinUnsealed,
		CID:           cid,
		Path:          folderPath,
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestFilecoinUnsealedSuite(t *testing.T) {
	suite.Run(t, new(FilecoinUnsealedSuite))
}

// Before all suite
func (suite *FilecoinUnsealedSuite) SetupAllSuite() {

}

// Before each test
func (suite *FilecoinUnsealedSuite) SetupTest() {
	var setupErr error
	cm = system.NewCleanupManager()
	ctx = context.Background()
	tempDir = suite.T().TempDir()
	driver, setupErr = NewStorage(cm, filepath.Join(tempDir, "{{.CID}}"))
	require.NoError(suite.T(), setupErr)
}

func (suite *FilecoinUnsealedSuite) TearDownTest() {
}

func (suite *FilecoinUnsealedSuite) TearDownAllSuite() {

}

func (suite *FilecoinUnsealedSuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
}

func (suite *FilecoinUnsealedSuite) TestHasStorageLocally() {
	cid := "123"
	spec := suite.prepareCid(cid)
	hasStorageTrue, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorageTrue, "file that exists should return true for HasStorageLocally")
	spec.CID = "apples"
	hasStorageFalse, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.False(suite.T(), hasStorageFalse, "file that does not exist should return false for HasStorageLocally")
}

func (suite *FilecoinUnsealedSuite) TestGetVolumeSize() {
	cid := "123"
	fileContents := "hello world"
	spec := suite.prepareCid(cid)
	filePath := filepath.Join(spec.Path, "file")
	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	require.NoError(suite.T(), err)
	volumeSize, err := driver.GetVolumeSize(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(len(fileContents)), volumeSize, "the volume size should be the size of the file")
}

func (suite *FilecoinUnsealedSuite) TestPrepareStorage() {
	cid := "123"
	spec := suite.prepareCid(cid)
	volume, err := driver.PrepareStorage(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), spec.Path, volume.Source, "the volume source should be the same as the spec path")
}

func (suite *FilecoinUnsealedSuite) TestExplode() {
	cid := "123"
	spec := suite.prepareCid(cid)
	exploded, err := driver.Explode(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(exploded), 1, "the exploded list should be 1 item long")
	require.Equal(suite.T(), exploded[0].CID, cid, "the cid is correct")
}
