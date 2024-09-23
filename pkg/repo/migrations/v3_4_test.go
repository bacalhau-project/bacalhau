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
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/config_legacy"
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

// repo resulting from `bacalhau serve --node-type=compute,requester`
func (suite *V3MigrationsTestSuite) TestV3MigrationWithFullRepo() {
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_full_repo")
	suite.copyRepo(testDataPath)
	// validate the repo copy is correct
	suite.DirExists(filepath.Join(suite.TempDir, "compute_store"))
	suite.FileExists(filepath.Join(suite.TempDir, "compute_store", "executions.db"))
	suite.FileExists(filepath.Join(suite.TempDir, "compute_store", "n-321fd9bf-3a7c-45f5-9b6b-fb9725ac646d.registration.lock"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages", "bacalhau-local-publisher"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store", "nats-store"))
	suite.FileExists(filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "plugins"))
	suite.FileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.FileExists(filepath.Join(suite.TempDir, "update.json"))
	suite.FileExists(filepath.Join(suite.TempDir, "user_id.pem"))
	configPath := filepath.Join(suite.TempDir, config.DefaultFileName)

	// define a config in the repo we are migrating with the correct paths defined in it
	// based on the old directory structure.
	executionStorePath := filepath.Join(suite.TempDir, "compute_store", "executions.db")
	jobStorePath := filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db")
	expectedInstallationID := "12345678-abcd-1234-abcd-123456789012"
	tokensPath := filepath.Join(suite.TempDir, "tokens.json")
	f, err := os.Create(tokensPath)
	suite.Require().NoError(err)
	defer f.Close()
	_, err = createConfig(configPath, fmt.Sprintf(`
Node:
    ClientAPI:
# these are values that will be migrated to the new config API.Address field
        Host: 1.2.3.4
        Port: 9999
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
	suite.FileExists(configPath)

	// verify the repo's current version is 3
	repoVersion3, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version3, repoVersion3)

	// open the repo to trigger the migration to version 4
	suite.Require().NoError(err)
	suite.Require().NoError(suite.repo.Open())

	// verify the repo's new current version is 4
	repoVersion4, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version4, repoVersion4)

	// verify the config file remains in the repo directory
	suite.FileExists(configPath)
	c, err := config.New(config.WithPaths(configPath))
	suite.Require().NoError(err)
	var bacCfg types.Bacalhau
	suite.Require().NoError(c.Unmarshal(&bacCfg))
	suite.Require().Equal("1.2.3.4", bacCfg.API.Host)
	suite.Require().Equal(9999, bacCfg.API.Port)

	// verify database files are present
	suite.FileExists(filepath.Join(suite.TempDir, types.OrchestratorDirName, types.JobStoreFileName))
	suite.FileExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionStoreFileName))

	// verify old files were removed
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))

	// verify the new files exist
	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))

	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator", "nats-store"))
	suite.FileExists(filepath.Join(suite.TempDir, "orchestrator", "state_boltdb.db"))

	// old compute directories were replaced with new ones
	suite.NoDirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionDirName))

	sysmeta, err := suite.repo.SystemMetadata()
	suite.Require().NoError(err)

	// verify we can read the expected last update time from it.
	suite.Require().Equal(time.UnixMilli(0).UTC(), sysmeta.LastUpdateCheck.UTC())

	// the node name was migrated from the old config to system_metadata.yaml
	suite.Require().Equal("n-321fd9bf-3a7c-45f5-9b6b-fb9725ac646d", sysmeta.NodeName)
}

// repo resulting from `bacalhau version`
func (suite *V3MigrationsTestSuite) TestV3MigrationWithMinimalRepo() {
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_minimal_repo")
	suite.copyRepo(testDataPath)
	// validate the repo copy is correct
	suite.DirExists(filepath.Join(suite.TempDir, "compute_store"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "compute_store", "executions.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store", "nats-store"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "plugins"))
	suite.FileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.FileExists(filepath.Join(suite.TempDir, "update.json"))
	suite.FileExists(filepath.Join(suite.TempDir, "user_id.pem"))

	// verify the repo's current version is 3
	repoVersion3, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version3, repoVersion3)

	// open the repo to trigger the migration to version 4
	suite.Require().NoError(err)
	suite.Require().NoError(suite.repo.Open())

	// verify the repo's new current version is 4
	repoVersion4, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version4, repoVersion4)

	// verify the config file hasn't been created
	configPath := filepath.Join(suite.TempDir, config_legacy.FileName)
	suite.NoFileExists(configPath)

	suite.NoFileExists(filepath.Join(suite.TempDir, types.OrchestratorDirName, types.JobStoreFileName))
	suite.NoFileExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionStoreFileName))
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "orchestrator", "state_boltdb.db"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator", "nats-store"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "executor_storages"))

	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator"))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionDirName))

	sysmeta, err := suite.repo.SystemMetadata()
	suite.Require().NoError(err)

	suite.Require().Equal(time.UnixMilli(0).UTC(), sysmeta.LastUpdateCheck.UTC())

	// check that the migration doesn't create a node name if a config file wasn't present.
	suite.Require().Empty(sysmeta.NodeName)
}

// repo resulting from `bacalhau serve --node-type=requester`
func (suite *V3MigrationsTestSuite) TestV3MigrationWithOrchestratorRepo() {
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_orchestrator_repo")
	suite.copyRepo(testDataPath)
	// validate the repo copy is correct
	suite.DirExists(filepath.Join(suite.TempDir, "compute_store"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "compute_store", "executions.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages", "bacalhau-local-publisher"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store", "nats-store"))
	suite.FileExists(filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "plugins"))
	suite.FileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.FileExists(filepath.Join(suite.TempDir, "update.json"))
	suite.FileExists(filepath.Join(suite.TempDir, "user_id.pem"))
	configPath := filepath.Join(suite.TempDir, config.DefaultFileName)

	// define a config in the repo we are migrating with the correct paths defined in it
	// based on the old directory structure.
	executionStorePath := filepath.Join(suite.TempDir, "compute_store", "executions.db")
	jobStorePath := filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db")
	expectedInstallationID := "12345678-abcd-1234-abcd-123456789012"
	tokensPath := filepath.Join(suite.TempDir, "tokens.json")
	f, err := os.Create(tokensPath)
	suite.Require().NoError(err)
	defer f.Close()
	_, err = createConfig(configPath, fmt.Sprintf(`
Node:
    ClientAPI:
# these are values that will be migrated to the new config API.Address field
        Host: 1.2.3.4
        Port: 9999
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
	suite.FileExists(configPath)

	// verify the repo's current version is 3
	repoVersion3, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version3, repoVersion3)

	// open the repo to trigger the migration to version 4
	suite.Require().NoError(err)
	suite.Require().NoError(suite.repo.Open())

	// verify the repo's new current version is 4
	repoVersion4, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version4, repoVersion4)

	// verify the config file remains in the repo directory and contains the migrated values
	suite.FileExists(configPath)
	c, err := config.New(config.WithPaths(configPath))
	suite.Require().NoError(err)
	var bacCfg types.Bacalhau
	suite.Require().NoError(c.Unmarshal(&bacCfg))
	suite.Require().Equal("1.2.3.4", bacCfg.API.Host)
	suite.Require().Equal(9999, bacCfg.API.Port)

	// verify database files are present
	suite.FileExists(filepath.Join(suite.TempDir, types.OrchestratorDirName, types.JobStoreFileName))
	//suite.FileExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionStoreFileName))

	// verify old file were removed
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))

	// verify the new files exists
	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))

	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator", "nats-store"))
	suite.FileExists(filepath.Join(suite.TempDir, "orchestrator", "state_boltdb.db"))

	// old compute directories were replaced with new ones
	suite.NoDirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionDirName))

	sysmeta, err := suite.repo.SystemMetadata()
	suite.Require().NoError(err)

	// verify we can read the expected last update time from it.
	suite.Require().Equal(time.UnixMilli(0).UTC(), sysmeta.LastUpdateCheck.UTC())

	// the node name was migrated from the old config to system_metadata.yaml
	suite.Require().Equal("n-321fd9bf-3a7c-45f5-9b6b-fb9725ac646d", sysmeta.NodeName)
}

// repo resulting from `bacalhau serve --node-type=compute --orchestrators=bootstrap.production.bacalhau.org`
func (suite *V3MigrationsTestSuite) TestV3MigrationWithComputeRepo() {
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_compute_repo")
	suite.copyRepo(testDataPath)
	// validate the repo copy is correct
	suite.DirExists(filepath.Join(suite.TempDir, "compute_store"))
	suite.FileExists(filepath.Join(suite.TempDir, "compute_store", "executions.db"))
	suite.FileExists(filepath.Join(suite.TempDir, "compute_store", "n-ee26c2a9-9ff3-4ccc-a224-ffc0a1da7d24.registration.lock"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, "executor_storages", "bacalhau-local-publisher"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store", "nats-store"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db"))
	suite.DirExists(filepath.Join(suite.TempDir, "plugins"))
	suite.FileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.FileExists(filepath.Join(suite.TempDir, "update.json"))
	suite.FileExists(filepath.Join(suite.TempDir, "user_id.pem"))
	configPath := filepath.Join(suite.TempDir, config.DefaultFileName)

	// define a config in the repo we are migrating with the correct paths defined in it
	// based on the old directory structure.
	executionStorePath := filepath.Join(suite.TempDir, "compute_store", "executions.db")
	jobStorePath := filepath.Join(suite.TempDir, "orchestrator_store", "jobs.db")
	expectedInstallationID := "12345678-abcd-1234-abcd-123456789012"
	tokensPath := filepath.Join(suite.TempDir, "tokens.json")
	f, err := os.Create(tokensPath)
	suite.Require().NoError(err)
	defer f.Close()
	_, err = createConfig(configPath, fmt.Sprintf(`
Node:
    ClientAPI:
# these are values that will be migrated to the new config API.Address field
        Host: 1.2.3.4
        Port: 9999
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
	suite.FileExists(configPath)

	// verify the repo's current version is 3
	repoVersion3, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version3, repoVersion3)

	// open the repo to trigger the migration to version 4
	suite.Require().NoError(err)
	suite.Require().NoError(suite.repo.Open())

	// verify the repo's new current version is 4
	repoVersion4, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.Version4, repoVersion4)

	// verify the config file remains in the repo directory and contains the migrated values
	suite.FileExists(configPath)
	c, err := config.New(config.WithPaths(configPath))
	suite.Require().NoError(err)
	var bacCfg types.Bacalhau
	suite.Require().NoError(c.Unmarshal(&bacCfg))
	suite.Require().Equal("1.2.3.4", bacCfg.API.Host)
	suite.Require().Equal(9999, bacCfg.API.Port)

	// verify database files are present
	suite.NoFileExists(filepath.Join(suite.TempDir, types.OrchestratorDirName, types.JobStoreFileName))
	suite.FileExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionStoreFileName))

	// verify old file were removed
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))

	// verify the new files exists
	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))

	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator_store"))
	suite.DirExists(filepath.Join(suite.TempDir, "orchestrator"))
	suite.NoDirExists(filepath.Join(suite.TempDir, "orchestrator", "nats-store"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "orchestrator", "state_boltdb.db"))

	sysmeta, err := suite.repo.SystemMetadata()
	suite.Require().NoError(err)
	// old compute directories were replaced with new ones
	suite.NoDirExists(filepath.Join(suite.TempDir, "executor_storages"))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName))
	suite.DirExists(filepath.Join(suite.TempDir, types.ComputeDirName, types.ExecutionDirName))

	// verify we can read the expected last update time from it.
	suite.Require().Equal(time.UnixMilli(0).UTC(), sysmeta.LastUpdateCheck.UTC())

	// the node name was migrated from the old config to system_metadata.yaml
	suite.Require().Equal("n-321fd9bf-3a7c-45f5-9b6b-fb9725ac646d", sysmeta.NodeName)
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
