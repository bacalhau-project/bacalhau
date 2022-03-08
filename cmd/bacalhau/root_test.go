package bacalhau

import (
	"os"
	"testing"

	testutils "github.com/filecoin-project/bacalhau/internal/test"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type RootSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

func (suite *RootSuite) SetupSuite() {
}

func (suite *RootSuite) SetupTest() {
	os.Setenv("DEBUG", "true")
	suite.rootCmd = RootCmd
}

// Default hello world for bacalhau - execute with no arguments
func (suite *RootSuite) Test_RootHelloWorld() {
	viper.Reset()

	_, out, err, _ := testutils.ExecuteCommandC(suite.T(), suite.rootCmd, "")

	// First line of the help text
	assert.NoError(suite.T(), err, "Error when calling command line with no arguments")
	assert.Contains(suite.T(), string(out), "Compute over data")

}

func TestRootSuite(t *testing.T) {
	suite.Run(t, new(RootSuite))
}
