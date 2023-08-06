//go:build unit || !integration

package touchfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

type touchFsSuite struct {
	suite.Suite

	testDir string
}

func TestTouchFSSuite(t *testing.T) {
	suite.Run(t, new(touchFsSuite))
}

func (suite *touchFsSuite) SetupTest() {
	testDir := suite.T().TempDir()
	suite.testDir = testDir

	file, err := os.Create(filepath.Join(testDir, "test.txt"))
	suite.Require().NoError(err)

	_, err = file.WriteString("hello")
	suite.Require().NoError(err)
	file.Close()
	logger.ConfigureTestLogging(suite.T())
}

func (suite *touchFsSuite) TearDownTest() {
	err := os.RemoveAll(suite.testDir)
	suite.Require().NoError(err)
}

func (suite *touchFsSuite) TestExistingFile() {
	touchFs := New(suite.testDir)
	contents, err := fs.ReadFile(touchFs, "test.txt")
	suite.Require().NoError(err)
	suite.Require().Equal("hello", string(contents))
}

func (suite *touchFsSuite) TestNewFile() {
	touchFs := New(suite.testDir)
	contents, err := fs.ReadFile(touchFs, "new.txt")
	suite.Require().NoError(err)
	suite.Require().Equal("", string(contents))
	suite.Require().FileExists(filepath.Join(suite.testDir, "new.txt"))
}

func (suite *touchFsSuite) TestWritingToNewFile() {
	touchFs := New(suite.testDir)
	file, err := touchFs.Open("new.txt")
	suite.Require().NoError(err)
	defer file.Close()

	writer, ok := file.(io.Writer)
	suite.Require().True(ok)

	i, err := io.WriteString(writer, "cool")
	suite.Require().NoError(err)
	suite.Require().Equal(4, i)

	suite.Require().FileExists(filepath.Join(suite.testDir, "new.txt"))

	contents, err := os.ReadFile(filepath.Join(suite.testDir, "new.txt"))
	suite.Require().NoError(err)
	suite.Require().Equal("cool", string(contents))
}
