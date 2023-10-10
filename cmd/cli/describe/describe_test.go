//go:build unit || !integration

package describe_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

// Define the suite, and absorb the built-in basic suite
// functionality from testify - including a T() method which
// returns the current testing context
type DescribeSuite struct {
	cmdtesting.BaseSuite
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
						j := testutils.MakeJobWithOpts(s.T(),
							jobutils.WithEngineSpec(
								model.NewEngineBuilder().
									WithType(strings.ToLower(model.EngineNoop.String())).
									WithParam("Entrypoint-Unique-Array", uuid.NewString()).
									Build(),
							),
						)
						job, err := s.Client.Submit(ctx, &j)
						s.Require().NoError(err)
						submittedJob = job // Default to the last job submitted, should be fine?
					}
				}
				returnedJobDescription := &model.JobWithInfo{}

				// No job id (should error)
				_, _, err := cmdtesting.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
				)
				s.Require().Error(err, "Submitting a describe request with no id should error.")

				// Job Id at the end
				_, out, err := cmdtesting.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
					submittedJob.Metadata.ID,
				)
				s.Require().NoError(err, "Error in describing job: %+v", err)

				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				s.Require().NoError(err, "Error in unmarshalling description: %+v", err)

				s.Require().Equal(submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")

				submittedJobEngineSpec, err := submittedJob.Spec.EngineSpec.Serialize()
				s.Require().NoError(err)
				returnedJobEngineSpec, err := returnedJobDescription.Job.Spec.EngineSpec.Serialize()
				s.Require().NoError(err)
				s.Require().Equal(
					submittedJobEngineSpec,
					returnedJobEngineSpec,
					fmt.Sprintf("Submitted job entrypoints not the same as the description. expected: %+v, received: %+v",
						submittedJob.Spec.EngineSpec, returnedJobDescription.Job.Spec.EngineSpec))

				// Job Id in the middle
				_, out, err = cmdtesting.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					submittedJob.Metadata.ID,
					"--api-port", fmt.Sprint(s.Port),
				)

				s.Require().NoError(err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				s.Require().NoError(err, "Error in unmarshalling description: %+v", err)
				s.Require().Equal(submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")

				returnedJobEngineSpec, err = returnedJobDescription.Job.Spec.EngineSpec.Serialize()
				s.Require().NoError(err)
				s.Require().Equal(
					submittedJobEngineSpec,
					returnedJobEngineSpec,
					fmt.Sprintf("Submitted job entrypoints not the same as the description. %d - %d - %s - %d", tc.numberOfAcceptNodes, tc.numberOfRejectNodes, tc.jobState, n.numOfJobs))

				// Short job id
				_, out, err = cmdtesting.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					idgen.ShortID(submittedJob.Metadata.ID),
					"--api-port", fmt.Sprint(s.Port),
				)

				s.Require().NoError(err, "Error in describing job: %+v", err)
				err = model.YAMLUnmarshalWithMax([]byte(out), returnedJobDescription)
				s.Require().NoError(err, "Error in unmarshalling description: %+v", err)
				s.Require().Equal(submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")

				returnedJobEngineSpec, err = returnedJobDescription.Job.Spec.EngineSpec.Serialize()
				s.Require().NoError(err)
				s.Require().Equal(
					submittedJobEngineSpec,
					returnedJobEngineSpec,
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

			j := testutils.MakeNoopJob(s.T())
			job, err := s.Client.Submit(ctx, j)
			s.Require().NoError(err)
			submittedJob = job // Default to the last job submitted, should be fine?

			var returnedJob = &model.Job{}

			var args []string

			args = append(args, "describe", "--api-host", s.Host, "--api-port", fmt.Sprint(s.Port), submittedJob.Metadata.ID)
			if tc.includeEvents {
				args = append(args, "--include-events")
			}

			// Job Id at the end
			_, out, err := cmdtesting.ExecuteTestCobraCommand(args...)
			s.Require().NoError(err, "Error in describing job: %+v", err)

			err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJob)
			s.Require().NoError(err, "Error in unmarshalling description: %+v", err)

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

	util.Fatal = util.FakeFatalErrorHandler
	for _, tc := range tests {
		for _, n := range numOfJobsTests {
			func() {

				var submittedJob *model.Job
				ctx := context.Background()

				for i := 0; i < n.numOfJobs; i++ {
					j := testutils.MakeJobWithOpts(s.T(),
						jobutils.WithEngineSpec(
							model.NewEngineBuilder().
								WithType(strings.ToLower(model.EngineNoop.String())).
								WithParam("Entrypoint-Unique-Array", uuid.NewString()).
								Build(),
						),
					)
					jj, err := s.Client.Submit(ctx, &j)
					s.Require().Nil(err)
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

				_, out, err = cmdtesting.ExecuteTestCobraCommand("describe",
					"--api-host", s.Host,
					"--api-port", fmt.Sprint(s.Port),
					jobID,
				)
				if tc.describeIDEdgecase == "" {
					s.Require().NoError(err, "Error in describing job: %+v", err)

					err = model.YAMLUnmarshalWithMax([]byte(out), &returnedJobDescription)
					s.Require().NoError(err, "Error in unmarshalling description: %+v", err)
					s.Require().Equal(submittedJob.Metadata.ID, returnedJobDescription.Job.Metadata.ID, "IDs do not match.")

					submittedJobEngineSpec, err := submittedJob.Spec.EngineSpec.Serialize()
					s.Require().NoError(err)
					returnedJobEngineSpec, err := returnedJobDescription.Job.Spec.EngineSpec.Serialize()
					s.Require().NoError(err)
					s.Require().Equal(
						submittedJobEngineSpec,
						returnedJobEngineSpec,
						fmt.Sprintf("Submitted job entrypoints not the same as the description. Edgecase: %s", tc.describeIDEdgecase))
				} else {
					c := &model.TestFatalErrorHandlerContents{}
					s.Require().NoError(model.JSONUnmarshalWithMax([]byte(out), &c))
					e := bacerrors.NewJobNotFound(tc.describeIDEdgecase)
					s.Require().Contains(c.Message, e.GetMessage(), "Job not found error string not found.", err)
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
