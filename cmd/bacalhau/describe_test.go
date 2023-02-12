//go:build unit || !integration

package bacalhau

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/bacerrors"
	"github.com/filecoin-project/bacalhau/pkg/model"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DescribeSuite struct {
	BaseSuite
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

				for i := 0; i < tc.numberOfAcceptNodes; i++ {
					for k := 0; k < n.numOfJobs; k++ {
						j := testutils.MakeNoopJob()
						j.Spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
						s, err := suite.client.Submit(ctx, j)
						require.NoError(suite.T(), err)
						submittedJob = s // Default to the last job submitted, should be fine?
					}
				}
				returnedJobDescription := &model.JobWithInfo{}

				// No job id (should error)
				_, out, err := ExecuteTestCobraCommand(suite.T(), "describe",
					"--api-host", suite.host,
					"--api-port", suite.port,
				)
				require.Error(suite.T(), err, "Submitting a describe request with no id should error.")

				// Job Id at the end
				_, out, err = ExecuteTestCobraCommand(suite.T(), "describe",
					"--api-host", suite.host,
					"--api-port", suite.port,
					submittedJob.Metadata.ID,
				)
				require.NoError(suite.T(), err, "Error in describing job: %+v", err)

				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)

				require.Equal(suite.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Job Id in the middle
				_, out, err = ExecuteTestCobraCommand(suite.T(), "describe",
					"--api-host", suite.host,
					submittedJob.Metadata.ID,
					"--api-port", suite.port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Short job id
				_, out, err = ExecuteTestCobraCommand(suite.T(), "describe",
					"--api-host", suite.host,
					submittedJob.Metadata.ID[0:model.ShortIDLength],
					"--api-port", suite.port,
				)

				require.NoError(suite.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(suite.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(suite.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(suite.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
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

			j := testutils.MakeNoopJob()
			s, err := suite.client.Submit(ctx, j)
			require.NoError(suite.T(), err)
			submittedJob = s // Default to the last job submitted, should be fine?

			var returnedJob = &model.Job{}

			var args []string

			args = append(args, "describe", "--api-host", suite.host, "--api-port", suite.port, submittedJob.Metadata.ID)
			if tc.includeEvents {
				args = append(args, "--include-events")
			}

			// Job Id at the end
			_, out, err := ExecuteTestCobraCommand(suite.T(), args...)
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

				for i := 0; i < n.numOfJobs; i++ {
					j := testutils.MakeNoopJob()
					j.Spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
					jj, err := s.client.Submit(ctx, j)
					require.Nil(s.T(), err)
					submittedJob = jj // Default to the last job submitted, should be fine?
				}

				var returnedJobDescription = &model.JobWithInfo{}
				var err error
				var out string
				var jobID string

				// If describeID is empty, should return use submitted ID. Otherwise, use describeID
				if tc.describeIDEdgecase == "" {
					jobID = submittedJob.Metadata.ID
				} else {
					jobID = tc.describeIDEdgecase
				}

				_, out, err = ExecuteTestCobraCommand(s.T(), "describe",
					"--api-host", s.host,
					"--api-port", s.port,
					jobID,
				)
				if tc.describeIDEdgecase == "" {
					require.NoError(s.T(), err, "Error in describing job: %+v", err)

					err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJobDescription)
					require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)
					require.Equal(s.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
					require.Equal(s.T(),
						submittedJob.Spec.Docker.Entrypoint[0],
						returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
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
