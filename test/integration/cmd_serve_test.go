package integration_test

import (
	"fmt"
	"os"
	"testing"

	cmd "github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/mocks"
	"github.com/filecoin-project/bacalhau/pkg/utils"
	"github.com/spf13/cobra"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type ServeSuite struct {
	suite.Suite
	serveCmd       *cobra.Command
}

func (suite *ServeSuite) SetupAllSuite() {

}

// before each test
func (suite *ServeSuite) SetupTest() {
	os.Setenv("TEST_PASS", "1")
	suite.serveCmd = cmd.ServeCmd
}

func (suite *ServeSuite) Test_DefaultServe() {
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
	command, out, err := utils.ExecuteCommandC(suite.serveCmd, logger, "serve",  "--", mocks.CALL_RUN_BACALHAU_RPC_SERVER_SUCCESSFUL_PROBE)

	// Putting empty assignments here for debugging in the future
	_ = command
	_ = err
	_ = out

	assert.NoError(suite.T(), err, fmt.Sprintf("Error found (non expected): %v", err))
}

func TestServeSuite(t *testing.T) {
	suite.Run(t, new(ServeSuite))
}