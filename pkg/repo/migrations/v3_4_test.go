//go:build unit || !integration

package migrations

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type V3MigrationsTestSuite struct {
	BaseMigrationTestSuite // Embed the base suite
	repo                   *repo.FsRepo
}

func (suite *V3MigrationsTestSuite) SetupTest() {
	suite.BaseMigrationTestSuite.SetupTest()
	migrations, err := repo.NewMigrationManager(
		V3Migration,
	)
	suite.Require().NoError(err)

	suite.repo, err = repo.NewFS(repo.FsRepoParams{
		Path:       suite.TempDir,
		Migrations: migrations,
	})
}

func TestV3MigrationsTestSuite(t *testing.T) {
	suite.Run(t, new(V3MigrationsTestSuite))
}

func (suite *V3MigrationsTestSuite) TestV3MigrationWithDefaultRepo() {
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_defaults")
	suite.copyRepo(testDataPath)
	configPath := filepath.Join(suite.TempDir, config.FileName)

	// define a config in the repo we are migrating with the correct paths defined in it
	executionStorePath := filepath.Join(suite.TempDir, repo.ComputeDirKey, "executions.db")
	jobStorePath := filepath.Join(suite.TempDir, repo.OrchestratorDirKey, "jobs.db")
	expectedInstallationID := "12345678-abcd-1234-abcd-123456789012"
	tokensPath := filepath.Join(suite.TempDir, "tokens.json")
	f, err := os.Create(tokensPath)
	suite.Require().NoError(err)
	defer f.Close()
	_, err = createConfig(configPath, fmt.Sprintf(`
Node:
    Compute:
        ExecutionStore:
            Type: BoltDB
            Path: %s
    Name: n-321fd9bf-3a7c-45f5-9b6b-fb9725ac646d
    Requester:
        JobStore:
            Type: BoltDB
            Path: %s
User:
    InstallationID: %s
Auth:
    TokensPath: %s
`, executionStorePath, jobStorePath, expectedInstallationID, tokensPath))
	suite.Require().NoError(err)

	// verify the repo's current version is 3
	repoVersion3, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version3, repoVersion3)

	// open the repo to trigger the migration to version 4
	cfg, err := config.New()
	suite.Require().NoError(err)
	suite.Require().NoError(suite.repo.Open(cfg))

	// verify the repo's new current version is 4
	repoVersion4, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version4, repoVersion4)

	// verify the config file remains present
	suite.FileExists(filepath.Join(suite.TempDir, config.FileName))

	// verify database files are present
	suite.FileExists(filepath.Join(suite.TempDir, repo.OrchestratorDirKey, "jobs.db"))
	suite.FileExists(filepath.Join(suite.TempDir, repo.ComputeDirKey, "executions.db"))

	// verify old file were removed
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))

	// verify the new files exists
	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))

	// old directories were replaced with new ones
	suite.NoDirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, repo.ComputeDirKey, "executions"))

	// verify we can read the expected installationID from it.
	actualInstallationID, err := suite.repo.ReadInstallationID()
	suite.Require().NoError(err)
	suite.Require().Equal(expectedInstallationID, actualInstallationID)

	// verify we can read the expected last update time from it.
	actualLastUpdateCheck, err := suite.repo.ReadLastUpdateCheck()
	suite.Require().NoError(err)
	suite.Require().Equal(time.UnixMilli(0).UTC(), actualLastUpdateCheck.UTC())
}

// createConfig creates a config file with the given content
func createConfig(path string, content string) (*os.File, error) {
	tmpfile, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}
	return tmpfile, nil
}
