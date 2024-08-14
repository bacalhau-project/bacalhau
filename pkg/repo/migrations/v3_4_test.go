//go:build unit || !integration

package migrations

import (
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
	expectedInstallationID := "12345678-abcd-1234-abcd-123456789012"
	// Copy test data to the suite's temporary directory
	testDataPath := filepath.Join("testdata", "v3_defaults")
	suite.copyRepo(testDataPath)

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

	// verify old file were removed
	suite.NoFileExists(filepath.Join(suite.TempDir, "repo.version"))
	suite.NoFileExists(filepath.Join(suite.TempDir, "update.json"))

	// verify the new file exists
	suite.FileExists(filepath.Join(suite.TempDir, "system_metadata.yaml"))
	// verify we can read the expected installationID from it.
	actualInstallationID, err := suite.repo.ReadInstallationID()
	suite.Require().NoError(err)
	suite.Require().Equal(expectedInstallationID, actualInstallationID)
	// verify we can read the expected last update time from it.
	actualLastUpdateCheck, err := suite.repo.ReadLastUpdateCheck()
	suite.Require().NoError(err)
	//
	suite.Require().Equal(time.UnixMilli(0).UTC(), actualLastUpdateCheck.UTC())
}
