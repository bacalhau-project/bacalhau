//go:build !integration

package bacalhau

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DescribeSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before each test
func (suite *DescribeSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
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
				var submittedJob *model.Job
				ctx := context.Background()
				c, cm := publicapi.SetupRequesterNodeForTests(suite.T())
				defer cm.Cleanup()

				for i := 0; i < tc.numberOfAcceptNodes; i++ {
					for k := 0; k < n.numOfJobs; k++ {
						j := publicapi.MakeNoopJob()
						j.Spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
						s, err := c.Submit(ctx, j, nil)
						require.NoError(suite.T(), err)
						submittedJob = s // Default to the last job submitted, should be fine?
					}
				}

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				returnedJob := &model.Job{}

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

				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJob)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJob.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJob.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Job Id in the middle
				_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					submittedJob.ID,
					"--api-port", port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJob)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJob.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJob.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Short job id
				_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "describe",
					"--api-host", host,
					submittedJob.ID[0:model.ShortIDLength],
					"--api-port", port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJob)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.ID, returnedJob.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJob.Spec.Docker.Entrypoint[0],
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
			var submittedJob *model.Job
			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(suite.T())
			defer cm.Cleanup()

			j := publicapi.MakeNoopJob()
			s, err := c.Submit(ctx, j, nil)
			require.NoError(suite.T(), err)
			submittedJob = s // Default to the last job submitted, should be fine?

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			var returnedJob = &model.Job{}

			var args []string

			args = append(args, "describe", "--api-host", host, "--api-port", port, submittedJob.ID)
			if tc.includeEvents {
				args = append(args, "--include-events")
			}

			// Job Id at the end
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, args...)
			require.NoError(suite.T(), err, "Error in describing job: %+v", err)

			err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJob)
			require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)

			// TODO: #600 When we figure out how to add events to a noop job, uncomment the below
			// require.True(suite.T(), eventsWereIncluded == tc.includeEvents,
			// 	fmt.Sprintf("Events include: %v\nExpected: %v", eventsWereIncluded, tc.includeEvents))

			// require.True(suite.T(), localEventsWereIncluded == tc.includeEvents,
			// 	fmt.Sprintf("Events included: %v\nExpected: %v", localEventsWereIncluded, tc.includeEvents))

		}()
	}

}

func (s *DescribeSuite) TestDescribeJobEdgeCases() {
	tests := []struct {
		describeIDEdgecase string
		errorMessage       string
	}{
		{describeIDEdgecase: "", errorMessage: ""},
		{describeIDEdgecase: "BAD_JOB_ID", errorMessage: "No job ID found."},
	}

	numOfJobsTests := []struct {
		numOfJobs int
	}{
		{numOfJobs: 1}, // just enough that describe could get screwed up
	}

	for _, tc := range tests {
		for _, n := range numOfJobsTests {
			func() {
				Fatal = FakeFatalErrorHandler

				var submittedJob *model.Job
				ctx := context.Background()
				c, cm := publicapi.SetupRequesterNodeForTests(s.T())
				defer cm.Cleanup()

				for i := 0; i < n.numOfJobs; i++ {
					j := publicapi.MakeNoopJob()
					j.Spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
					jj, err := c.Submit(ctx, j, nil)
					require.Nil(s.T(), err)
					submittedJob = jj // Default to the last job submitted, should be fine?
				}

				parsedBasedURI, _ := url.Parse(c.BaseURI)
				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				var returnedJob = model.NewJob()
				var err error
				var out string
				var jobID string

				// If describeID is empty, should return use submitted ID. Otherwise, use describeID
				if tc.describeIDEdgecase == "" {
					jobID = submittedJob.ID
				} else {
					jobID = tc.describeIDEdgecase
				}

				_, out, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "describe",
					"--api-host", host,
					"--api-port", port,
					jobID,
				)
				if tc.describeIDEdgecase == "" {
					require.NoError(s.T(), err, "Error in describing job: %+v", err)

					err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJob)
					require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)
					require.Equal(s.T(), submittedJob.ID, returnedJob.ID, "IDs do not match.")
					require.Equal(s.T(),
						submittedJob.Spec.Docker.Entrypoint[0],
						returnedJob.Spec.Docker.Entrypoint[0],
						fmt.Sprintf("Submitted job entrypoints not the same as the description. Edgecase: %s", tc.describeIDEdgecase))
				} else {
					c := &model.TestFatalErrorHandlerContents{}
					model.JSONUnmarshalWithMax([]byte(out), &c)
					e := bacerrors.NewJobNotFound(tc.describeIDEdgecase)
					require.Contains(s.T(), c.Message, e.GetMessage(), "Job not found error string not found.", err)
				}

			}()
		}
	}

}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDescribeSuite(t *testing.T) {
	suite.Run(t, new(DescribeSuite))
}
