//go:build unit || !integration

package migrations

import (
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
)

type BaseMigrationTestSuite struct {
	suite.Suite
	TempDir string // Temporary directory for testing
}

// SetupTest runs before each test in the suite.
func (suite *BaseMigrationTestSuite) SetupTest() {
	viper.Reset()
	suite.TempDir = suite.T().TempDir() // Create a temporary directory for testing
}

// copyRepo copies source repo to the suite's temporary directory.
func (suite *BaseMigrationTestSuite) copyRepo(src string) {
	suite.T().Logf("copying repo from %s to %s", src, suite.TempDir)
	suite.copyDir(src, suite.TempDir)
}

// copyDir copies the contents of the src directory into the dst directory.
// Utilizes suite.Require().NoError for immediate failure on error.
func (suite *BaseMigrationTestSuite) copyDir(src, dst string) {
	srcInfo, err := os.Stat(src)
	suite.Require().NoError(err)

	err = os.MkdirAll(dst, srcInfo.Mode())
	suite.Require().NoError(err)

	entries, err := os.ReadDir(src)
	suite.Require().NoError(err)

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			suite.copyDir(srcPath, dstPath) // Recursively copy subdirectories
		} else {
			suite.copyFile(srcPath, dstPath) // Copy files
		}
	}
}

// copyFile copies a single file from src to dst.
// Utilizes suite.Require().NoError for immediate failure on error.
func (suite *BaseMigrationTestSuite) copyFile(src, dst string) {
	srcFile, err := os.Open(src)
	suite.Require().NoError(err)
	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst)
	suite.Require().NoError(err)
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	suite.Require().NoError(err)

	srcInfo, err := srcFile.Stat()
	suite.Require().NoError(err)

	err = os.Chmod(dst, srcInfo.Mode())
	suite.Require().NoError(err)
}
