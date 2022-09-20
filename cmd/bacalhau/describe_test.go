package bacalhau

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
}

func (suite *DescribeSuite) TearDownTest() {
}

func (suite *DescribeSuite) TearDownAllSuite() {

}

func (suite *DescribeSuite) TestDescribeJob() {
	tests := []struct {
		numberOfAcceptNodes int
		numberOfRejectNodes int
		jobState            string
	}{
		{numberOfAcceptNodes: 1, numberOfRejectNodes: 0, jobState: model.JobEventResultsPublished.String()}, // Run and accept
		{numberOfAcceptNodes: 2, numberOfRejectNodes: 0, jobState: model.JobEventResultsPublished.String()}, // Run and accept
		{numberOfAcceptNodes: 1, numberOfRejectNodes: 1, jobState: model.JobEventResultsPublished.String()}, // Run and accept
	}

	numOfJobsTests := []struct {
		numOfJobs int
	}{
		{numOfJobs: 1},
		{numOfJobs: 21}, // one more than the default list length
	}

	for _, tc := range tests {
		for _, n := range numOfJobsTests {
			func() {
				var submittedJob model.Job
				ctx := context.Background()
				c, cm := publicapi.SetupTests(suite.T())
				defer cm.Cleanup()

				for i := 0; i < tc.numberOfAcceptNodes; i++ {
					for i := 0; i < n.numOfJobs; i++ {
						spec, deal := publicapi.MakeNoopJob()
						spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
						s, err := c.Submit(ctx, spec, deal, nil)
						require.NoError(suite.T(), err)
						submittedJob = s // Default to the last job submitted, should be fine?
					}
				}

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				var returnedJobDescription = &jobDescription{}

				// No job id (should error)
				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					"--api-port", port,
				)
				require.Error(suite.T(), err, "Submitting a describe request with no id should error.")

				// Job Id at the end
				_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					"--api-port", port,
					submittedJob.ID,
				)
				require.NoError(suite.T(), err, "Error in describing job: %+v", err)

				err = yaml.Unmarshal([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJobDescription.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Job Id in the middle
				_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					submittedJob.ID,
					"--api-port", port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = yaml.Unmarshal([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJobDescription.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Short job id
				_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					submittedJob.ID[0:6],
					"--api-port", port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = yaml.Unmarshal([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJobDescription.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

			}()
		}
	}

}

func (suite *DescribeSuite) TestDescribeJobIncludeEvents() {
	tests := []struct {
		includeEvents bool
	}{
		{includeEvents: false},
		{includeEvents: true},
	}

	for _, tc := range tests {
		func() {
			var submittedJob model.Job
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			spec, deal := publicapi.MakeNoopJob()
			s, err := c.Submit(ctx, spec, deal, nil)
			require.NoError(suite.T(), err)
			submittedJob = s // Default to the last job submitted, should be fine?

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			var returnedJobDescription = &jobDescription{}

			var args []string

			args = append(args, "describe", "--api-host", host, "--api-port", port, submittedJob.ID)
			if tc.includeEvents {
				args = append(args, "--include-events")
			}

			// Job Id at the end
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
			require.NoError(suite.T(), err, "Error in describing job: %+v", err)

			err = yaml.Unmarshal([]byte(out), returnedJobDescription)
			require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)

			// TODO: #600 When we figure out how to add events to a noop job, uncomment the below
			// require.True(suite.T(), eventsWereIncluded == tc.includeEvents,
			// 	fmt.Sprintf("Events include: %v\nExpected: %v", eventsWereIncluded, tc.includeEvents))

			// require.True(suite.T(), localEventsWereIncluded == tc.includeEvents,
			// 	fmt.Sprintf("Events included: %v\nExpected: %v", localEventsWereIncluded, tc.includeEvents))

		}()
	}

}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDescribeSuite(t *testing.T) {
	suite.Run(t, new(DescribeSuite))
}
