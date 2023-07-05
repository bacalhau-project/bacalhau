//go:build unit || !integration

package describe_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/testing"
	util2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/util"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DescribeSuite struct {
	cmdtesting2.BaseSuite
}

func (s *DescribeSuite) TestDescribeJob() {
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
						job, err := s.Client.Submit(ctx, j)
						require.NoError(s.T(), err)
						submittedJob = job // Default to the last job submitted, should be fine?
					}
				}
				returnedJobDescription := &model.JobWithInfo{}

				// No job id (should error)
				_, _, err := cmdtesting2.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
				)
				require.Error(s.T(), err, "Submitting a describe request with no id should error.")

				// Job Id at the end
				_, out, err := cmdtesting2.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
					submittedJob.Metadata.ID,
				)
				require.NoError(s.T(), err, "Error in describing job: %+v", err)

				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)

				require.Equal(s.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(s.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Job Id in the middle
				_, out, err = cmdtesting2.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					submittedJob.Metadata.ID,
					"--api-port", fmt.Sprint(s.Port),
				)

				require.NoError(s.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(s.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(s.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Short job id
				_, out, err = cmdtesting2.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					submittedJob.Metadata.ID[0:model.ShortIDLength],
					"--api-port", fmt.Sprint(s.Port),
				)

				require.NoError(s.T(), err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)
				require.Equal(s.T(), submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")
				require.Equal(s.T(),
					submittedJob.Spec.Docker.Entrypoint[0],
					returnedJobDescription.Job.Spec.Docker.Entrypoint[0],
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

			}()
		}
	}

}

func (s *DescribeSuite) TestDescribeJobIncludeEvents() {
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
			job, err := s.Client.Submit(ctx, j)
			require.NoError(s.T(), err)
			submittedJob = job // Default to the last job submitted, should be fine?

			var returnedJob = &model.Job{}

			var args []string

			args = append(args, "describe", "--api-host", s.Host, "--api-port", fmt.Sprint(s.Port), submittedJob.Metadata.ID)
			if tc.includeEvents {
				args = append(args, "--include-events")
			}

			// Job Id at the end
			_, out, err := cmdtesting2.ExecuteTestCobraCommand(args...)
			require.NoError(s.T(), err, "Error in describing job: %+v", err)

			err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJob)
			require.NoError(s.T(), err, "Error in unmarshalling description: %+v", err)

			// TODO: #600 When we figure out how to add events to a noop job, uncomment the below
			// require.True(s.T(), eventsWereIncluded == tc.includeEvents,
			// 	fmt.Sprintf("Events include: %v\nExpected: %v", eventsWereIncluded, tc.includeEvents))

			// require.True(s.T(), localEventsWereIncluded == tc.includeEvents,
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

	util2.Fatal = util2.FakeFatalErrorHandler
	for _, tc := range tests {
		for _, n := range numOfJobsTests {
			func() {

				var submittedJob *model.Job
				ctx := context.Background()

				for i := 0; i < n.numOfJobs; i++ {
					j := testutils.MakeNoopJob()
					j.Spec.Docker.Entrypoint = []string{"Entrypoint-Unique-Array", uuid.NewString()}
					jj, err := s.Client.Submit(ctx, j)
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

				_, out, err = cmdtesting2.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
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
					s.NoError(model.JSONUnmarshalWithMax([]byte(out), &c))
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
