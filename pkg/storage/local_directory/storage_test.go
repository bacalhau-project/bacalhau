//go:build unit || !integration

package localdirectory

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var ctx context.Context
var tempDir string
var driver *StorageProvider
var cm *system.CleanupManager

type LocalDirectorySuite struct {
	suite.Suite
}

func (suite *LocalDirectorySuite) prepareStorageSpec(sourcePath string) model.StorageSpec {
	folderPath := filepath.Join(tempDir, sourcePath)
	err := os.MkdirAll(folderPath, os.ModePerm)
	require.NoError(suite.T(), err)
	return model.StorageSpec{
		// source path is some kind of sub-path
		// inside our local folder
		SourcePath: sourcePath,
		Path:       "/path/inside/the/container",
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestLocalDirectorySuite(t *testing.T) {
	suite.Run(t, new(LocalDirectorySuite))
}

// Before each test
func (suite *LocalDirectorySuite) SetupTest() {
	var setupErr error
	logger.ConfigureTestLogging(suite.T())
	cm = system.NewCleanupManager()
	ctx = context.Background()
	tempDir = suite.T().TempDir()
	driver, setupErr = NewStorage(cm, tempDir)
	require.NoError(suite.T(), setupErr)
}

func (suite *LocalDirectorySuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
}

func (suite *LocalDirectorySuite) TestHasStorageLocally() {
	subpath := "apples/oranges"
	spec := suite.prepareStorageSpec(subpath)
	hasStorageTrue, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorageTrue, "file that exists should return true for HasStorageLocally")
	spec.SourcePath = "apples/pears"
	hasStorageFalse, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.False(suite.T(), hasStorageFalse, "file that does not exist should return false for HasStorageLocally")
}

func (suite *LocalDirectorySuite) TestGetVolumeSize() {
	subpath := "apples/oranges"
	folderPath := filepath.Join(tempDir, subpath)
	fileContents := "hello world"
	spec := suite.prepareStorageSpec(subpath)
	filePath := filepath.Join(folderPath, "file")
	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	require.NoError(suite.T(), err)
	volumeSize, err := driver.GetVolumeSize(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(len(fileContents)), volumeSize, "the volume size should be the size of the file")
}

func (suite *LocalDirectorySuite) TestPrepareStorage() {
	subpath := "apples/oranges"
	spec := suite.prepareStorageSpec(subpath)
	volume, err := driver.PrepareStorage(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), volume.Source, filepath.Join(tempDir, subpath), "the volume source is correct")
}

func (suite *LocalDirectorySuite) TestExplode() {
	subpath := "apples/oranges"
	spec := suite.prepareStorageSpec(subpath)
	exploded, err := driver.Explode(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), len(exploded), 1, "the exploded list should be 1 item long")
	require.Equal(suite.T(), exploded[0].SourcePath, subpath, "the subpath is correct")
}
