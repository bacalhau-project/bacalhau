//go:build unit || !integration

package localdirectory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/local"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type LocalDirectorySuite struct {
	suite.Suite
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
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(tc.allowedPaths)})
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
			allowedPaths:      []string{filepath.Join(tmpDir, "*")},
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
			allowedPaths:      []string{"**"},
			hasStorageLocally: true,
		},
		{
			name:              "pattern",
			sourcePath:        tmpFile,
			allowedPaths:      []string{filepath.Join(tmpDir, "file*")},
			hasStorageLocally: true,
		},
		{
			name:              "pattern with suffix",
			sourcePath:        tmpFile,
			allowedPaths:      []string{filepath.Join(tmpDir, "file*1")},
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
		{
			name:              "has rw permission",
			sourcePath:        tmpFile,
			allowedPaths:      []string{tmpFile + ":rw"},
			hasStorageLocally: true,
		},
		{
			name:              "has and requires rw permission",
			sourcePath:        tmpFile + ":rw",
			allowedPaths:      []string{tmpFile + ":rw"},
			hasStorageLocally: true,
		},
		{
			name:              "missing rw permission",
			sourcePath:        tmpFile + ":rw",
			allowedPaths:      []string{tmpFile},
			hasStorageLocally: false,
		},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(tc.allowedPaths)})
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
			allowedPaths:       []string{filepath.Join(tmpDir, "*")},
			expectedVolumeSize: 0,
		},
		{
			name:               "size of file2",
			sourcePath:         file2,
			allowedPaths:       []string{filepath.Join(tmpDir, "*")},
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
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(tc.allowedPaths)})
			require.NoError(s.T(), err)

			volumeSize, err := storageProvider.GetVolumeSize(context.Background(), s.prepareStorageSpec(tc.sourcePath))
			if tc.shouldFail {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.expectedVolumeSize, volumeSize)
		})
	}
}

func (s *LocalDirectorySuite) TestPrepareStorage() {
	tmpDir := s.T().TempDir()
	folderPath := filepath.Join(tmpDir, "sub", "path")
	s.Require().NoError(os.MkdirAll(folderPath, 0755))
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths([]string{filepath.Join(tmpDir, "**:rw")})})
	require.NoError(s.T(), err)

	for _, tc := range []struct {
		name      string
		readWrite bool
	}{
		{name: "readonly", readWrite: false},
		{name: "readWrite", readWrite: true},
	} {
		s.Run(tc.name, func() {
			path := folderPath
			if tc.readWrite {
				path += ":rw"
			}
			spec := s.prepareStorageSpec(path)
			volume, err := storageProvider.PrepareStorage(context.Background(), spec)
			require.NoError(s.T(), err)
			require.Equal(s.T(), volume.Source, folderPath)
			require.Equal(s.T(), volume.Target, spec.Mount)
			require.Equal(s.T(), volume.ReadOnly, !tc.readWrite)
			require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
		})
	}

}

func (s *LocalDirectorySuite) prepareStorageSpec(sourcePath string) spec.Storage {
	readWrite := false
	if strings.HasSuffix(sourcePath, ":rw") {
		readWrite = true
		sourcePath = strings.TrimSuffix(sourcePath, ":rw")
	}
	out, err := (&local.LocalStorageSpec{Source: sourcePath, ReadWrite: readWrite}).AsSpec("TODO", "/path/inside/the/container")
	s.Require().NoError(err)
	return out
}
