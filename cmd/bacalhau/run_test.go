package bacalhau

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type RunSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *RunSuite) SetupAllSuite() {

}

// Before each test
func (suite *RunSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
	suite.rootCmd = RootCmd
}

func (suite *RunSuite) TearDownTest() {
}

func (suite *RunSuite) TearDownAllSuite() {

}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRunSuite(t *testing.T) {
	suite.Run(t, new(RunSuite))
}
