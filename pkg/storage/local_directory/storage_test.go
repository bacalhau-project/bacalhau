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
	err := os.MkdirAll(sourcePath, os.ModePerm)
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
	driver = NewStorageProvider(StorageProviderParams{AllowedPaths: []string{tempDir}})
	require.NoError(suite.T(), setupErr)
}

func (suite *LocalDirectorySuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
}

func (suite *LocalDirectorySuite) TestHasStorageLocally() {
	folderPath := filepath.Join(tempDir, "apples/oranges")
	spec := suite.prepareStorageSpec(folderPath)
	hasStorageTrue, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorageTrue, "file that exists should return true for HasStorageLocally")
	spec.SourcePath = "apples/pears"
	hasStorageFalse, err := driver.HasStorageLocally(ctx, spec)
	require.NoError(suite.T(), err)
	require.False(suite.T(), hasStorageFalse, "file that does not exist should return false for HasStorageLocally")
}

func (suite *LocalDirectorySuite) TestGetVolumeSize() {
	folderPath := filepath.Join(tempDir, "apples/oranges")
	fileContents := "hello world"
	spec := suite.prepareStorageSpec(folderPath)
	filePath := filepath.Join(folderPath, "file")
	err := os.WriteFile(filePath, []byte(fileContents), 0644)
	require.NoError(suite.T(), err)
	volumeSize, err := driver.GetVolumeSize(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), uint64(len(fileContents)), volumeSize, "the volume size should be the size of the file")
}

func (suite *LocalDirectorySuite) TestPrepareStorage() {
	folderPath := filepath.Join(tempDir, "apples/oranges")
	spec := suite.prepareStorageSpec(folderPath)
	volume, err := driver.PrepareStorage(ctx, spec)
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), volume.Source, folderPath, "the volume source is correct")
}
