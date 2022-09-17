package system

import (
	"testing"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemUtilsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemUtilsSuite(t *testing.T) {
	suite.Run(t, new(SystemUtilsSuite))
}

// Before all suite
func (suite *SystemUtilsSuite) SetupAllSuite() {

}

// Before each test
func (suite *SystemUtilsSuite) SetupTest() {
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemUtilsSuite) TearDownTest() {
}

func (suite *SystemUtilsSuite) TearDownAllSuite() {

}
