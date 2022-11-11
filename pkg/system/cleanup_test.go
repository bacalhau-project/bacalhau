//go:build !integration

package system

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type SystemCleanupSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestSystemCleanupSuite(t *testing.T) {
	suite.Run(t, new(SystemCleanupSuite))
}

// Before each test
func (suite *SystemCleanupSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), InitConfigForTesting())
}

func (suite *SystemCleanupSuite) TestCleanupManager() {
	clean := false

	cm := NewCleanupManager()
	cm.RegisterCallback(func() error {
		clean = true
		return nil
	})

	cm.Cleanup()
	require.True(suite.T(), clean, "cleanup handler failed to run registered functions")
}
