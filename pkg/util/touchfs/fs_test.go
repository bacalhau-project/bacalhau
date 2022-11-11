//go:build !integration

package touchfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
	require.NoError(suite.T(), err)

	_, err = file.WriteString("hello")
	require.NoError(suite.T(), err)
	file.Close()
	logger.ConfigureTestLogging(suite.T())
}

func (suite *touchFsSuite) TearDownTest() {
	err := os.RemoveAll(suite.testDir)
	require.NoError(suite.T(), err)
}

func (suite *touchFsSuite) TestExistingFile() {
	touchFs := New(suite.testDir)
	contents, err := fs.ReadFile(touchFs, "test.txt")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "hello", string(contents))
}

func (suite *touchFsSuite) TestNewFile() {
	touchFs := New(suite.testDir)
	contents, err := fs.ReadFile(touchFs, "new.txt")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "", string(contents))
	require.FileExists(suite.T(), filepath.Join(suite.testDir, "new.txt"))
}

func (suite *touchFsSuite) TestWritingToNewFile() {
	touchFs := New(suite.testDir)
	file, err := touchFs.Open("new.txt")
	require.NoError(suite.T(), err)
	defer file.Close()

	writer, ok := file.(io.Writer)
	require.True(suite.T(), ok)

	i, err := io.WriteString(writer, "cool")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), 4, i)

	require.FileExists(suite.T(), filepath.Join(suite.testDir, "new.txt"))

	contents, err := os.ReadFile(filepath.Join(suite.testDir, "new.txt"))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "cool", string(contents))
}
