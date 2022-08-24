package filecoin_unsealed

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var ctx context.Context
var tempDir string
var driver *StorageProvider
var cm *system.CleanupManager

type FilecoinUnsealedSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestFilecoinUnsealedSuite(t *testing.T) {
	suite.Run(t, new(FilecoinUnsealedSuite))
}

// Before all suite
func (suite *FilecoinUnsealedSuite) SetupAllSuite() {

}

// Before each test
func (suite *FilecoinUnsealedSuite) SetupTest() {
	var setupErr error
	cm = system.NewCleanupManager()
	ctx = context.Background()
	tempDir, setupErr = ioutil.TempDir("", "bacalhau-filecoin-unsealed-test")
	require.NoError(suite.T(), setupErr)
	driver, setupErr = NewStorageProvider(cm, fmt.Sprintf("%s/{{.Cid}}", tempDir))
	require.NoError(suite.T(), setupErr)
}

func (suite *FilecoinUnsealedSuite) TearDownTest() {
}

func (suite *FilecoinUnsealedSuite) TearDownAllSuite() {

}

func (suite *FilecoinUnsealedSuite) TestIsInstalled() {
	installed, err := driver.IsInstalled(ctx)
	require.NoError(suite.T(), err)
	require.True(suite.T(), installed)
}

func (suite *FilecoinUnsealedSuite) TestHasStorageLocally() {
	cid := "123"
	folderPath := fmt.Sprintf("%s/%s", tempDir, cid)
	err := os.MkdirAll(folderPath, os.ModePerm)
	require.NoError(suite.T(), err)
	hasStorageTrue, err := driver.HasStorageLocally(ctx, storage.StorageSpec{
		Engine: storage.StorageSourceFilecoinUnsealed,
		Cid:    cid,
	})
	require.NoError(suite.T(), err)
	require.True(suite.T(), hasStorageTrue, "file that exists should return true for HasStorageLocally")
	hasStorageFalse, err := driver.HasStorageLocally(ctx, storage.StorageSpec{
		Engine: storage.StorageSourceFilecoinUnsealed,
		Cid:    "apples",
	})
	require.NoError(suite.T(), err)
	require.False(suite.T(), hasStorageFalse, "file that does not exist should return false for HasStorageLocally")
}
