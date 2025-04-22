//go:build unit || !integration

package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type LocalStorageSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestLocalStorageSuite(t *testing.T) {
	suite.Run(t, new(LocalStorageSuite))
}

// Before the suite
func (s *LocalStorageSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
}

func (s *LocalStorageSuite) TestIsInstalled() {
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

func (s *LocalStorageSuite) TestHasStorageLocally() {
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
			name:              "file doesn't exist and does not allow read-write permission, dirver does not allow read-write permission",
			sourcePath:        filepath.Join(tmpDir, "unknown:ro"),
			allowedPaths:      []string{tmpDir + "/*"},
			hasStorageLocally: false,
		},
		{
			name:              "file doesn't exist and allows read-write permission, driver allows read-write permission",
			sourcePath:        filepath.Join(tmpDir, "unknown:rw"),
			allowedPaths:      []string{tmpDir + "/*:rw"},
			hasStorageLocally: true,
		},
		{
			name:              "file doesn't exist and allows read-write permission, dirver does not allow read-write permission",
			sourcePath:        filepath.Join(tmpDir, "unknown:rw"),
			allowedPaths:      []string{tmpDir + "/*:ro"},
			hasStorageLocally: false,
		},
		{
			name:              "file exists and does not allow read-write permission, driver allows read-write permission",
			sourcePath:        tmpFile,
			allowedPaths:      []string{tmpFile + ":rw"},
			hasStorageLocally: true,
		},
		{
			name:              "file exists and allows read-write permission, driver allows read-write permission",
			sourcePath:        tmpFile + ":rw",
			allowedPaths:      []string{tmpFile + ":rw"},
			hasStorageLocally: true,
		},
		{
			name:              "file exists and allows read-write permission, driver does not allow read-write permission",
			sourcePath:        tmpFile + ":rw",
			allowedPaths:      []string{tmpFile},
			hasStorageLocally: false,
		},
		{
			name:              "folder does not exist and allows read-write permission, driver allows read-write permission",
			sourcePath:        filepath.Join(tmpDir, "nonexistent_folder") + ":rw",
			allowedPaths:      []string{tmpDir + "/*:rw"},
			hasStorageLocally: true,
		},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(tc.allowedPaths)})
			require.NoError(s.T(), err)

			hasStorageLocally, err := storageProvider.HasStorageLocally(context.Background(), s.prepareStorageSpec(tc.sourcePath, ""))
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.hasStorageLocally, hasStorageLocally)
		})
	}
}

func (s *LocalStorageSuite) TestGetVolumeSize() {
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
		errorMessage       string
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
			errorMessage: "not allowlisted",
		},
		{
			name:         "no allowed paths",
			sourcePath:   file2,
			allowedPaths: []string{},
			shouldFail:   true,
			errorMessage: "is not allowlisted",
		},
		{
			name:         "missing rw permission",
			sourcePath:   file1 + ":rw",
			allowedPaths: []string{file1},
			shouldFail:   true,
			errorMessage: "is not granted write access",
		},
		{
			name:         "file doesn't exist, read-write permission not allowed",
			sourcePath:   filepath.Join(tmpDir, "unknown"),
			allowedPaths: []string{tmpDir + "/*"},
			shouldFail:   true,
			errorMessage: "does not exist and read-write access is not allowed",
		},
		{
			name:         "file doesn't exist, read-write permission is allowed",
			sourcePath:   filepath.Join(tmpDir, "unknown") + ":rw",
			allowedPaths: []string{tmpDir + "/*:rw"},
			shouldFail:   false,
		},
		{
			name:         "directory doesn't exist, read-write permission not allowed",
			sourcePath:   filepath.Join(tmpDir, "unknown_dir/"),
			allowedPaths: []string{tmpDir + "/*"},
			shouldFail:   true,
			errorMessage: "does not exist and read-write access is not allowed",
		},
		{
			name:         "directory doesn't exist, read-write permission is allowed",
			sourcePath:   filepath.Join(tmpDir, "unknown_dir/") + ":rw",
			allowedPaths: []string{tmpDir + "/*:rw"},
			shouldFail:   false,
		},
	} {
		s.Run(tc.name, func() {
			storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(tc.allowedPaths)})
			require.NoError(s.T(), err)

			volumeSize, err := storageProvider.GetVolumeSize(context.Background(), mock.Execution(), s.prepareStorageSpec(tc.sourcePath, ""))
			if tc.shouldFail {
				require.Error(s.T(), err)
				require.Truef(s.T(), strings.Contains(err.Error(), tc.errorMessage), "error message should contain %s, but got %s", tc.errorMessage, err.Error())
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), tc.expectedVolumeSize, volumeSize)
		})
	}
}

func (s *LocalStorageSuite) TestPrepareStorage_VolumeExists() {
	tmpDir := s.T().TempDir()
	existingFolder := filepath.Join(tmpDir, "sub", "path")
	s.Require().NoError(os.MkdirAll(existingFolder, 0755))

	allowedPaths := []string{filepath.Join(tmpDir, "**:rw")}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	for _, tc := range []struct {
		name       string
		sourcePath string
		isDir      bool
		shouldFail bool
	}{
		{
			name:       "folder exists and does not have read-write permission",
			sourcePath: existingFolder,
			shouldFail: false,
		},
		{
			name:       "folder exists and has read-write permission",
			sourcePath: existingFolder + ":rw",
			shouldFail: false,
		},
	} {
		s.Run(tc.name, func() {
			spec := s.prepareStorageSpec(tc.sourcePath, "")
			volume, err := storageProvider.PrepareStorage(context.Background(), s.T().TempDir(), mock.Execution(), spec)
			require.NoError(s.T(), err)
			require.Equal(s.T(), volume.Source, existingFolder)
			require.Equal(s.T(), volume.Target, spec.Target)
			require.Equal(s.T(), volume.ReadOnly, !s.readWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
		})
	}
}

func (s *LocalStorageSuite) TestPrepareStorage_VolumeDoesNotExist_CreateAsFile() {
	tmpDir := s.T().TempDir()
	nonExistingPath := filepath.Join(tmpDir, "sub", "nonexisting", "path")

	allowedPaths := []string{filepath.Join(tmpDir, "**:rw")}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	for _, tc := range []struct {
		name       string
		sourcePath string
		shouldFail bool
	}{
		{
			name:       "does not have read-write permission",
			sourcePath: nonExistingPath,
			shouldFail: true,
		},
		{
			name:       "has read-write permission",
			sourcePath: nonExistingPath + ":rw",
			shouldFail: false,
		},
		{
			name:       "source path looks like a directory but createAs is file",
			sourcePath: nonExistingPath + "/:rw", // A path with a trailing slash is considered a directory
			shouldFail: true,
		},
	} {
		s.Run(tc.name, func() {
			s.T().Cleanup(func() {
				os.RemoveAll(s.stripReadWrite(tc.sourcePath))
			})

			spec := s.prepareStorageSpec(tc.sourcePath, File.String())
			volume, err := storageProvider.PrepareStorage(context.Background(), s.T().TempDir(), mock.Execution(), spec)
			if tc.shouldFail {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), volume.Source, s.stripReadWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Target, spec.Target)
			require.Equal(s.T(), volume.ReadOnly, !s.readWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
			require.Truef(s.T(), s.isExistingFile(volume.Source), "file %s should exist", volume.Source)
		})
	}
}

func (s *LocalStorageSuite) TestPrepareStorage_VolumeDoesNotExist_CreateAsDir() {
	tmpDir := s.T().TempDir()
	nonExistingPath1 := filepath.Join(tmpDir, "sub", "nonexisting", "path")
	nonExistingPath2 := filepath.Join(tmpDir, "sub", "nonexisting", "path.txt")

	allowedPaths := []string{filepath.Join(tmpDir, "**:rw")}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	for _, tc := range []struct {
		name       string
		sourcePath string
		shouldFail bool
	}{
		{
			name:       "does not have read-write permission",
			sourcePath: nonExistingPath1,
			shouldFail: true,
		},
		{
			name:       "has read-write permission",
			sourcePath: nonExistingPath1 + ":rw",
			shouldFail: false,
		},
		{
			name:       "source path looks like a file but createAs is dir",
			sourcePath: nonExistingPath2 + ":rw",
			shouldFail: false,
		},
	} {
		s.Run(tc.name, func() {
			s.T().Cleanup(func() {
				os.RemoveAll(s.stripReadWrite(tc.sourcePath))
			})

			spec := s.prepareStorageSpec(tc.sourcePath, Dir.String())
			volume, err := storageProvider.PrepareStorage(context.Background(), s.T().TempDir(), mock.Execution(), spec)
			if tc.shouldFail {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), volume.Source, s.stripReadWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Target, spec.Target)
			require.Equal(s.T(), volume.ReadOnly, !s.readWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
			require.Truef(s.T(), s.isExistingDirectory(volume.Source), "directory %s should exist", volume.Source)
		})
	}
}

func (s *LocalStorageSuite) TestPrepareStorage_VolumeDoesNotExist_CreateAsInfer() {
	tmpDir := s.T().TempDir()
	directoryPath := filepath.Join(tmpDir, "sub", "nonexisting", "path") + "/"
	filePath := filepath.Join(tmpDir, "sub", "nonexisting", "path.txt")

	allowedPaths := []string{filepath.Join(tmpDir, "**:rw")}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	for _, tc := range []struct {
		name       string
		sourcePath string
		isDir      bool
		shouldFail bool
	}{
		{
			name:       "source path looks like a directory",
			sourcePath: directoryPath + ":rw",
			isDir:      true,
			shouldFail: false,
		},
		{
			name:       "source path looks like a file",
			sourcePath: filePath + ":rw",
			isDir:      false,
			shouldFail: false,
		},
	} {
		s.Run(tc.name, func() {
			s.T().Cleanup(func() {
				os.RemoveAll(s.stripReadWrite(tc.sourcePath))
			})

			spec := s.prepareStorageSpec(tc.sourcePath, Infer.String())
			volume, err := storageProvider.PrepareStorage(context.Background(), s.T().TempDir(), mock.Execution(), spec)
			if tc.shouldFail {
				require.Error(s.T(), err)
				return
			}
			require.NoError(s.T(), err)
			require.Equal(s.T(), volume.Source, s.stripReadWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Target, spec.Target)
			require.Equal(s.T(), volume.ReadOnly, !s.readWrite(tc.sourcePath))
			require.Equal(s.T(), volume.Type, storage.StorageVolumeConnectorBind)
			if tc.isDir {
				require.Truef(s.T(), s.isExistingDirectory(volume.Source), "directory %s should exist", volume.Source)
			} else {
				require.Truef(s.T(), s.isExistingFile(volume.Source), "file %s should exist", volume.Source)
			}

		})
	}
}

func (s *LocalStorageSuite) TestCleanupStorage() {
	tmpDir := s.T().TempDir()
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths([]string{filepath.Join(tmpDir, "**:rw")})})
	require.NoError(s.T(), err)

	storageVolume := storage.StorageVolume{
		Type:     storage.StorageVolumeConnectorBind,
		ReadOnly: false,
		Source:   tmpDir,
		Target:   "/path/inside/the/container",
	}

	err = storageProvider.CleanupStorage(context.Background(), s.prepareStorageSpec(tmpDir, ""), storageVolume)
	require.NoError(s.T(), err)

	// Check that the directory still exists (as it should not be removed)
	_, err = os.Stat(tmpDir)
	require.NoError(s.T(), err)
}

func (s *LocalStorageSuite) TestHasStorageLocally_LegacyName() {
	tmpDir := s.T().TempDir()
	allowedPaths := []string{tmpDir}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	hasStorageLocally, err := storageProvider.HasStorageLocally(context.Background(), s.prepareLegacyStorageSpec(tmpDir, ""))
	require.NoError(s.T(), err)
	require.True(s.T(), hasStorageLocally)
}

func (s *LocalStorageSuite) TestGetVolumeSize_LegacyName() {
	tmpDir := s.T().TempDir()
	file := filepath.Join(tmpDir, "file1")
	s.Require().NoError(os.WriteFile(file, []byte("1234"), 0644))

	allowedPaths := []string{tmpDir + "/*"}
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths(allowedPaths)})
	require.NoError(s.T(), err)

	volumeSize, err := storageProvider.GetVolumeSize(context.Background(), mock.Execution(), s.prepareLegacyStorageSpec(file, ""))
	require.NoError(s.T(), err)
	require.Equal(s.T(), uint64(0), volumeSize)
}

func (s *LocalStorageSuite) TestPrepareStorage_LegacyName() {
	tmpDir := s.T().TempDir()
	folderPath := filepath.Join(tmpDir, "sub", "path")
	s.Require().NoError(os.MkdirAll(folderPath, 0755))
	storageProvider, err := NewStorageProvider(StorageProviderParams{AllowedPaths: ParseAllowPaths([]string{filepath.Join(tmpDir, "**:rw")})})
	require.NoError(s.T(), err)

	path := folderPath + ":rw"
	spec := s.prepareLegacyStorageSpec(path, "")
	_, err = storageProvider.PrepareStorage(context.Background(), s.T().TempDir(), mock.Execution(), spec)
	require.NoError(s.T(), err)
}

func (s *LocalStorageSuite) prepareSourceSpec(sourcePath string, createAs string) Source {
	readWrite := false
	if s.readWrite(sourcePath) {
		readWrite = true
		sourcePath = strings.TrimSuffix(sourcePath, ":rw")
	}
	createStrategy, _ := CreateStrategyFromString(createAs)
	return Source{
		SourcePath: sourcePath,
		ReadWrite:  readWrite,
		CreateAs:   createStrategy,
	}
}

func (s *LocalStorageSuite) prepareStorageSpec(sourcePath string, createAs string) models.InputSource {
	source := s.prepareSourceSpec(sourcePath, createAs).ToMap()
	return models.InputSource{
		Source: &models.SpecConfig{
			Type:   models.StorageSourceLocal,
			Params: source,
		},
		Target: "/path/inside/the/container",
	}
}

// Create a spec using "localDirectory" provider name
func (s *LocalStorageSuite) prepareLegacyStorageSpec(sourcePath string, createAs string) models.InputSource {
	source := s.prepareSourceSpec(sourcePath, createAs)
	return models.InputSource{
		Source: &models.SpecConfig{
			Type:   models.StorageSourceLocalDirectory,
			Params: source.ToMap(),
		},
		Target: "/path/inside/the/container",
	}
}

func (s *LocalStorageSuite) readWrite(sourcePath string) bool {
	if strings.HasSuffix(sourcePath, ":rw") {
		return true
	}
	return false
}

func (s *LocalStorageSuite) stripReadWrite(sourcePath string) string {
	if strings.HasSuffix(sourcePath, ":rw") {
		return strings.TrimSuffix(sourcePath, ":rw")
	}
	return sourcePath
}

func (s *LocalStorageSuite) fileShouldNotExist(path string) bool {
	path = s.stripReadWrite(path)
	_, err := os.Stat(path)
	require.Error(s.T(), err)
	return os.IsNotExist(err)
}

func (s *LocalStorageSuite) isExistingFile(path string) bool {
	stat, err := os.Stat(path)
	require.NoError(s.T(), err)
	return stat.Mode().IsRegular()
}

func (s *LocalStorageSuite) isExistingDirectory(path string) bool {
	path = s.stripReadWrite(path)
	stat, err := os.Stat(path)
	require.NoError(s.T(), err)
	return stat.IsDir()
}
