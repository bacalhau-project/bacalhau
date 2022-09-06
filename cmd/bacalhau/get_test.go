package bacalhau

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestGetSuite(t *testing.T) {
	suite.Run(t, new(GetSuite))
}

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type GetSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *GetSuite) SetupAllSuite() {

}

// Before each test
func (suite *GetSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
}

func (suite *GetSuite) TearDownTest() {
}

func (suite *GetSuite) TearDownAllSuite() {

}

func (suite *GetSuite) TestGetJob() {
	const NumberOfNodes = 3

	numOfJobsTests := []struct {
		numOfJobs int
	}{
		{numOfJobs: 1},
		{numOfJobs: 21}, // one more than the default list length
	}

	outputDir, _ := os.MkdirTemp(os.TempDir(), "bacalhau-get-test-*")
	defer os.RemoveAll(outputDir)
	for _, n := range numOfJobsTests {
		func() {
			var submittedJob model.Job
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			for i := 0; i < NumberOfNodes; i++ {
				for i := 0; i < n.numOfJobs; i++ {
					spec, deal := publicapi.MakeGenericJob()
					s, err := c.Submit(ctx, spec, deal, nil)
					require.NoError(suite.T(), err)
					submittedJob = s // Default to the last job submitted, should be fine?
				}
			}

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)

			// No job id (should error)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "get",
				"--api-host", host,
				"--api-port", port,
			)
			require.Error(suite.T(), err, "Submitting a get request with no id should error.")

			outputDirWithID := fmt.Sprintf("%s/%s", outputDir, submittedJob.ID)
			os.Mkdir(outputDirWithID, util.OS_ALL_RWX)

			// Job Id at the end
			_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "get",
				"--api-host", host,
				"--api-port", port,
				"--output-dir", outputDirWithID,
				submittedJob.ID,
			)
			require.NoError(suite.T(), err, "Error in getting job: %+v", err)

			// Short Job ID
			_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "get",
				"--api-host", host,
				"--api-port", port,
				"--output-dir", outputDirWithID,
				submittedJob.ID[0:6],
			)
			require.NoError(suite.T(), err, "Error in getting short job: %+v", err)

			_ = out

		}()
	}

}
