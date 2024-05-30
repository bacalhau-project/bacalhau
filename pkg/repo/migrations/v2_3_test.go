//go:build unit || !integration

package migrations

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type V2MigrationsTestSuite struct {
	BaseMigrationTestSuite // Embed the base suite
	repo                   *repo.FsRepo
}

func (suite *V2MigrationsTestSuite) SetupTest() {
	suite.BaseMigrationTestSuite.SetupTest()
	migrations, err := repo.NewMigrationManager(
		V2Migration,
	)
	suite.Require().NoError(err)

	suite.repo, err = repo.NewFS(repo.FsRepoParams{
		Path:       suite.TempDir,
		Migrations: migrations,
	})
}

func TestV2MigrationsTestSuite(t *testing.T) {
	suite.Run(t, new(V2MigrationsTestSuite))
}

func (suite *V2MigrationsTestSuite) TestV2MigrationWithDefaultRepo() {
	libp2pPeerID := "QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV"

	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v2_defaults")
	suite.copyRepo(testDataPath)

	// verify the repo's current state
	suite.verifyInitialState(libp2pPeerID)

	// open the repo to trigger the migration
	suite.Require().NoError(suite.repo.Open(config.New()))

	repoVersion, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(expectedRepoVersion, repoVersion)

	// verify configs where updated as expected
	_, cfg, err := readConfig(*suite.repo)
	suite.Require().NoError(err)
	suite.Equal(filepath.Join(suite.TempDir, config.ComputeExecutionsStorePath), cfg.Node.Compute.ExecutionStore.Path)
	suite.Equal(filepath.Join(suite.TempDir, config.OrchestratorJobStorePath), cfg.Node.Requester.JobStore.Path)
	suite.Equal(libp2pPeerID, cfg.Node.Name)

	// verify the old directories were renamed
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-compute"))
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-requester"))
}

// TestV2MigrationWitCustomConfig test that migration with custom configs exist will not be modified
// and only execution and job store paths will be added
func (suite *V2MigrationsTestSuite) TestV2MigrationWitCustomConfig() {
	libp2pPeerID := "QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV"

	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v2_custom_configs")
	suite.copyRepo(testDataPath)

	// verify the repo's current state
	suite.verifyInitialState(libp2pPeerID)

	// open the repo to trigger the migration
	suite.Require().NoError(suite.repo.Open(config.New()))

	// verify the repo version was updated
	repoVersion, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(expectedRepoVersion, repoVersion)

	// verify configs where updated as expected, and that network port was not changed
	_, cfg, err := readConfig(*suite.repo)
	suite.Require().NoError(err)
	suite.Equal(filepath.Join(suite.TempDir, config.ComputeExecutionsStorePath), cfg.Node.Compute.ExecutionStore.Path)
	suite.Equal(filepath.Join(suite.TempDir, config.OrchestratorJobStorePath), cfg.Node.Requester.JobStore.Path)
	suite.Equal(libp2pPeerID, cfg.Node.Name)
	suite.Equal(123456789, cfg.Node.Network.Port)

	// verify the old directories were renamed
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-compute"))
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-requester"))
}

// TestV2MigrationWitCustomStores test that migration with custom stores will not be modified
func (suite *V2MigrationsTestSuite) TestV2MigrationWitCustomStores() {
	nodeName := "foo"
	libp2pPeerID := "QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV"

	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v2_custom_stores")
	suite.copyRepo(testDataPath)

	// verify the repo's current state
	suite.verifyInitialState(libp2pPeerID)

	// open the repo to trigger the migration
	suite.Require().NoError(suite.repo.Open(config.New()))

	// verify the repo version was updated
	repoVersion, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(expectedRepoVersion, repoVersion)

	// verify configs where NOT updated and the custom stores were not renamed
	_, cfg, err := readConfig(*suite.repo)
	suite.Require().NoError(err)
	suite.Equal("./QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV-compute/executions.db", cfg.Node.Compute.ExecutionStore.Path)
	suite.Equal("./QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV-requester/jobs.db", cfg.Node.Requester.JobStore.Path)
	suite.Equal(nodeName, cfg.Node.Name)
	suite.Equal(123456789, cfg.Node.Network.Port)

	// verify the old directories were NOT renamed
	suite.DirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-compute"))
	suite.DirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-requester"))
}

// TestV2MigrationWithEmptyStorePaths test that migration with store config exist, but with empty paths
func (suite *V2MigrationsTestSuite) TestV2MigrationWithEmptyStorePaths() {
	libp2pPeerID := "QmUBgU7xHKK44RuTHgrvnJfoSdZJS4fddT197iyTF5qjEV"

	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v2_empty_path")
	suite.copyRepo(testDataPath)

	// verify the repo's current state
	suite.verifyInitialState(libp2pPeerID)

	// open the repo to trigger the migration
	suite.Require().NoError(suite.repo.Open(config.New()))

	repoVersion, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(expectedRepoVersion, repoVersion)

	// verify configs where updated as expected
	_, cfg, err := readConfig(*suite.repo)
	suite.Require().NoError(err)
	suite.Equal(filepath.Join(suite.TempDir, config.ComputeExecutionsStorePath), cfg.Node.Compute.ExecutionStore.Path)
	suite.Equal(filepath.Join(suite.TempDir, config.OrchestratorJobStorePath), cfg.Node.Requester.JobStore.Path)
	suite.Equal(libp2pPeerID, cfg.Node.Name)

	// verify the old directories were renamed
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-compute"))
	suite.NoDirExists(filepath.Join(suite.TempDir, libp2pPeerID+"-requester"))
}

func (suite *V2MigrationsTestSuite) verifyInitialState(nodeID string) {
	repoVersion, err := suite.repo.Version()
	suite.Require().NoError(err)
	suite.Equal(repo.RepoVersion2, repoVersion)
	suite.DirExists(filepath.Join(suite.TempDir, nodeID+"-compute"))
	suite.DirExists(filepath.Join(suite.TempDir, nodeID+"-requester"))
}
