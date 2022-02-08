package integration_test

import (
	"testing"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/utils"
	"github.com/spf13/cobra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type RootSuite struct {
	suite.Suite
	rootCmd       *cobra.Command
}

// before each test
func (suite *RootSuite) SetupTest() {
	suite.rootCmd = cmd.RootCmd
}

func (suite *RootSuite) Test_DefaultRun() {
    logger, _ := utils.SetupLogsCapture()
    
    // logger.Warn("This is the warning")
    
    // if logs.Len() != 1 {
    //     suite.T().Errorf("No logs")
    // } else {
    //     entry := logs.All()[0]
    //     if entry.Level != zap.WarnLevel || entry.Message != "This is the warning" {
    //         suite.T().Errorf("Invalid log entry %v", entry)
    //     }
    // }
	command, out, err := utils.ExecuteCommandC(suite.rootCmd, logger, "")

	// Putting empty assignments here for debugging in the future
	_ = command
	_ = err

	assert.Contains(suite.T(), string(out), "bacalhau [command] --help")
}

func TestRootSuite(t *testing.T) {
	suite.Run(t, new(RootSuite))
}