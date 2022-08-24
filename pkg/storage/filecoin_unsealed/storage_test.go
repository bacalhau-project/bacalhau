package filecoin_unsealed

import (
	"context"
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type FilecoinUnsealedSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func FilecoinUnsealedSuiteSuite(t *testing.T) {
	suite.Run(t, new(FilecoinUnsealedSuite))
}

// Before all suite
func (suite *FilecoinUnsealedSuite) SetupAllSuite() {

}

// Before each test
func (suite *FilecoinUnsealedSuite) SetupTest() {

}

func (suite *FilecoinUnsealedSuite) TearDownTest() {
}

func (suite *FilecoinUnsealedSuite) TearDownAllSuite() {

}

func (suite *FilecoinUnsealedSuite) TestIsInstalled() {
	cm := system.NewCleanupManager()
	ctx := context.Background()
	driver, err := NewStorageProvider(cm, "")
	require.NoError(suite.T(), err)
	installed, err := driver.IsInstalled(ctx)
	require.True(suite.T(), installed)

}
