package bacalhau

import (
	"context"
	"net"
	"net/url"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DescribeSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before all suite
func (suite *DescribeSuite) SetupAllSuite() {

}

// Before each test
func (suite *DescribeSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
	suite.rootCmd = RootCmd
}

func (suite *DescribeSuite) TearDownTest() {
}

func (suite *DescribeSuite) TearDownAllSuite() {

}

func (suite *DescribeSuite) TestDescribeJob() {
	tableSortReverse = false

	tests := []struct {
		numberOfAcceptNodes int
		numberOfRejectNodes int
		jobState            string
	}{
		{numberOfAcceptNodes: 1, numberOfRejectNodes: 0, jobState: executor.JobStateComplete.String()}, // Run and accept
		// {numberOfJobs: 5, numberOfJobsOutput: 5},   // Test for 5 (less than default of 10)
		// {numberOfJobs: 20, numberOfJobsOutput: 10}, // Test for 20 (more than max of 10)
		// {numberOfJobs: 20, numberOfJobsOutput: 15}, // The default is 10 so test for non-default

	}

	for _, tc := range tests {
		func() {
			var submittedJob *executor.Job
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			for i := 0; i < tc.numberOfAcceptNodes; i++ {
				spec, deal := publicapi.MakeNoopJob()
				s, err := c.Submit(ctx, spec, deal, nil)
				assert.NoError(suite.T(), err)
				submittedJob = s
			}

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
				"--api-host", host,
				"--api-port", port,
				"-i", submittedJob.ID,
			)

			assert.NoError(suite.T(), err, "Error in describing job: %+v", err)

			var returnedJobDescription = &jobDescription{}
			err = yaml.Unmarshal([]byte(out), returnedJobDescription)

			assert.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)

			// assert.Equal(suite.T(), tc.numberOfJobsOutput, strings.Count(out, "\n"))

		}()
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDescribeSuite(t *testing.T) {
	suite.Run(t, new(DescribeSuite))
}
