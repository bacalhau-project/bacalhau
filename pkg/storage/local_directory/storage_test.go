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
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type LocalDirectorySuite struct {
	suite.Suite
	tempDir string
	driver  *StorageProvider
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestLocalDirectorySuite(t *testing.T) {
	suite.Run(t, new(LocalDirectorySuite))
}

// Before the suite
func (s *LocalDirectorySuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
}

// Before each test
func (s *LocalDirectorySuite) SetupTest() {
	var setupErr error
	s.tempDir = s.T().TempDir()
	s.driver, setupErr = NewStorageProvider(StorageProviderParams{AllowedPaths: []string{s.tempDir}})
	require.NoError(s.T(), setupErr)
}

func (s *LocalDirectorySuite) TestIsInstalled() {
	for _, tc := range []struct {
		name         string
		allowedPaths []string
		isInstalled  bool
	}{
		{name: "single allowed path", allowedPaths: []string{"tmp"}, isInstalled: true},
		{name: "asterisk allowed path", allowedPaths: []string{".*"}, isInstalled: true},
		{name: "multiple allowed paths", allowedPaths: []string{"tmp", "tmp2"}, isInstalled: true},
		{name: "no allowed paths", allowedPaths: []string{}, isInstalled: false},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: tc.allowedPaths})
			require.NoError(s.T(), err)

			installed, err := storageProvider.IsInstalled(context.Background())
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.isInstalled, installed)
		})
	}
}

func (s *LocalDirectorySuite) TestHasStorageLocally() {
	tmpDir := s.T().TempDir()
	tmpFile := filepath.Join(tmpDir, "file1")
	file, err := os.Create(tmpFile)
	s.Require().NoError(err)
	s.Require().NoError(file.Close())

	for _, tc := range []struct {
		name              string
		sourcePath        string
		allowedPaths      []string
		hasStorageLocally bool
	}{
		{
			name:              "path matches allowed path",
			sourcePath:        tmpFile,
			allowedPaths:      []string{tmpFile},
			hasStorageLocally: true,
		},
		{
			name:              "path descendant of an allowed path",
			sourcePath:        tmpFile,
			allowedPaths:      []string{tmpDir},
			hasStorageLocally: true,
		},
		{
			name:              "path is a directory",
			sourcePath:        tmpDir,
			allowedPaths:      []string{tmpDir},
			hasStorageLocally: true,
		},
		{
			name:              "asterisk allowed path",
			sourcePath:        tmpFile,
			allowedPaths:      []string{".*"},
			hasStorageLocally: true,
		},
		{
			name:              "path outside of allowed paths",
			sourcePath:        filepath.Dir(tmpDir),
			allowedPaths:      []string{tmpDir},
			hasStorageLocally: false,
		},
		{
			name:              "no allowed paths",
			allowedPaths:      []string{},
			hasStorageLocally: false,
		},
		{
			name:              "file doesn't exist",
			sourcePath:        filepath.Join(tmpDir, "unknown"),
			allowedPaths:      []string{".*"},
			hasStorageLocally: false,
		},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: tc.allowedPaths})
			require.NoError(s.T(), err)

			hasStorageLocally, err := storageProvider.HasStorageLocally(context.Background(), s.prepareStorageSpec(tc.sourcePath))
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.hasStorageLocally, hasStorageLocally)
		})
	}
}

func (s *LocalDirectorySuite) TestGetVolumeSize() {
	tmpDir := s.T().TempDir()
	file1 := filepath.Join(tmpDir, "file1")
	file2 := filepath.Join(tmpDir, "file2")
	s.Require().NoError(os.WriteFile(file1, []byte("1234"), 0644))
	s.Require().NoError(os.WriteFile(file2, []byte("12345678"), 0644))

	for _, tc := range []struct {
		name               string
		sourcePath         string
		allowedPaths       []string
		expectedVolumeSize uint64
		shouldFail         bool
	}{
		{
			name:               "size of file1",
			sourcePath:         file1,
			allowedPaths:       []string{tmpDir},
			expectedVolumeSize: 0,
		},
		{
			name:               "size of file2",
			sourcePath:         file2,
			allowedPaths:       []string{tmpDir},
			expectedVolumeSize: 0,
		},
		{
			name:               "size of parent directory",
			sourcePath:         tmpDir,
			allowedPaths:       []string{tmpDir},
			expectedVolumeSize: 0,
		},
		{
			name:         "path outside of allowed paths",
			sourcePath:   file2,
			allowedPaths: []string{file1},
			shouldFail:   true,
		},
		{
			name:         "no allowed paths",
			sourcePath:   file2,
			allowedPaths: []string{},
			shouldFail:   true,
		},
		{
			name:         "file doesn't exist",
			sourcePath:   filepath.Join(tmpDir, "unknown"),
			allowedPaths: []string{tmpDir},
			shouldFail:   true,
		},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: tc.allowedPaths})
			require.NoError(s.T(), err)

			volumeSize, err := storageProvider.GetVolumeSize(context.Background(), s.prepareStorageSpec(tc.sourcePath))
			if tc.shouldFail {
				require.Error(s.T(), err)
				return
			}
			require.Equal(s.T(), tc.expectedVolumeSize, volumeSize)
		})
	}
}

func (s *LocalDirectorySuite) TestPrepareStorage() {
	folderPath := filepath.Join(s.tempDir, "sub/path")
	spec := s.prepareStorageSpec(folderPath)
	volume, err := s.driver.PrepareStorage(context.Background(), spec)
	require.NoError(s.T(), err)
	require.Equal(s.T(), volume.Source, folderPath)
	require.Equal(s.T(), volume.Target, spec.Path)
	require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
}

func (s *LocalDirectorySuite) prepareStorageSpec(sourcePath string) model.StorageSpec {
	return model.StorageSpec{
		SourcePath: sourcePath,
		Path:       "/path/inside/the/container",
	}
}
