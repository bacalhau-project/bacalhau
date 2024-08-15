//go:build unit || !integration

package migrations

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func TestStagedMigration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-repo")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	viper.Set("repo", tempDir)
	// Create a test repo
	testRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       tempDir,
		Migrations: nil,
	})
	require.NoError(t, err, "Failed to create test repo")

	c, err := config.New()
	require.NoError(t, err)

	err = testRepo.Init(c)
	require.NoError(t, err)

	// Define a test migration function
	testMigration := func(r repo.FsRepo) error {
		path, err := r.Path()
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(path, "test_file.txt"), []byte("test content"), 0644)
	}

	// Create a staged migration
	migration := StagedMigration(1, 2, testMigration)

	// Perform the migration
	err = migration.Migrate(*testRepo)
	require.NoError(t, err, "Migration failed")

	// Check if the migration was successful
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	assert.FileExists(t, testFilePath, "Test file was not created")

	content, err := os.ReadFile(testFilePath)
	require.NoError(t, err, "Failed to read test file")

	assert.Equal(t, "test content", string(content), "Test file content is incorrect")
}

func TestPerformStagedMigration(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-repo")
	require.NoError(t, err, "Failed to create temp dir")
	defer os.RemoveAll(tempDir)

	// Create a test repo
	viper.Set("repo", tempDir)
	testRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       tempDir,
		Migrations: nil,
	})
	require.NoError(t, err, "Failed to create test repo")

	c, err := config.New()
	require.NoError(t, err)

	err = testRepo.Init(c)
	require.NoError(t, err)

	// Define a test migration function
	testMigration := func(r repo.FsRepo) error {
		path, err := r.Path()
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(path, "test_file.txt"), []byte("test content"), 0644)
	}

	// Perform the staged migration
	err = performStagedMigration(*testRepo, testMigration)
	require.NoError(t, err, "Staged migration failed")

	// Check if the migration was successful
	testFilePath := filepath.Join(tempDir, "test_file.txt")
	assert.FileExists(t, testFilePath, "Test file was not created")

	content, err := os.ReadFile(testFilePath)
	require.NoError(t, err, "Failed to read test file")

	assert.Equal(t, "test content", string(content), "Test file content is incorrect")
}

func TestCleanupStagingPath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-staging")
	require.NoError(t, err, "Failed to create temp dir")

	// Call cleanupStagingPath
	cleanupStagingPath(tempDir)

	// Check if the directory was removed
	_, err = os.Stat(tempDir)
	assert.True(t, os.IsNotExist(err), "Staging directory was not removed")
}

func TestCommitMigration(t *testing.T) {
	// Create temporary directories for testing
	stagingPath, err := os.MkdirTemp("", "test-staging")
	require.NoError(t, err, "Failed to create staging dir")
	defer os.RemoveAll(stagingPath)

	repoPath, err := os.MkdirTemp("", "test-repo")
	require.NoError(t, err, "Failed to create repo dir")
	defer os.RemoveAll(repoPath)

	// Create a test file in the staging path
	testFilePath := filepath.Join(stagingPath, "test_file.txt")
	err = os.WriteFile(testFilePath, []byte("test content"), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Commit the migration
	err = commitMigration(stagingPath, repoPath)
	require.NoError(t, err, "Commit migration failed")

	// Check if the staging directory was moved to the repo path
	_, err = os.Stat(stagingPath)
	assert.True(t, os.IsNotExist(err), "Staging directory was not removed")

	// Check if the test file exists in the repo path
	repoTestFilePath := filepath.Join(repoPath, "test_file.txt")
	assert.FileExists(t, repoTestFilePath, "Test file was not moved to repo path")

	// Check the content of the moved file
	content, err := os.ReadFile(repoTestFilePath)
	require.NoError(t, err, "Failed to read test file in repo")

	assert.Equal(t, "test content", string(content), "Test file content is incorrect")
}
