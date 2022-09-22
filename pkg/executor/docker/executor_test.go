package docker

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ExecutorDockerSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestExecutorSuiteSuite(t *testing.T) {
	suite.Run(t, new(ExecutorDockerSuite))
}

// Before all suite
func (suite *ExecutorDockerSuite) SetupAllSuite() {

}

// Before each test
func (suite *ExecutorDockerSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
}

func (suite *ExecutorDockerSuite) TearDownTest() {
}

func (suite *ExecutorDockerSuite) TearDownAllSuite() {

}
