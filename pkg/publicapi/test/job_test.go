//go:build unit || !integration

package test

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func (s *ServerSuite) TestJobOperations() {
	ctx := context.Background()
	job := mock.Job()
	putResponse, err := s.client.Jobs().Put(ctx, &apimodels.PutJobRequest{Job: job})
	s.Require().NoError(err)
	s.Require().NotNil(putResponse)
	s.Require().NotEmpty(putResponse.JobID)
	s.Require().NotEmpty(putResponse.EvaluationID)

	// retrieve the job
	getResponse, err := s.client.Jobs().Get(ctx, &apimodels.GetJobRequest{JobIDOrName: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(getResponse)
	s.Require().Equal(putResponse.JobID, getResponse.Job.ID)
	s.Require().EqualValues(job.Tasks, getResponse.Job.Tasks)

	// list the job
	listResponse, err := s.client.Jobs().List(ctx, &apimodels.ListJobsRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(listResponse)
	s.Require().NotEmpty(listResponse.Items)

	found := false
	for _, j := range listResponse.Items {
		if j.ID == putResponse.JobID {
			found = true
			break
		}
	}
	s.Require().True(found, "job %s not found in list", putResponse.JobID)

	// Wait for job executions to start, and for the job to complete
	s.Eventually(func() bool {
		res, err := s.client.Jobs().Get(ctx, &apimodels.GetJobRequest{JobIDOrName: putResponse.JobID})
		if err != nil {
			s.T().Logf("error getting job. will retry: %v", err)
			return false
		}
		if !res.Job.IsTerminal() {
			s.T().Logf("job is not terminal: %s. will retry", res.Job.State.StateType.String())
			return false
		}
		return true
	}, 5*time.Second, 50*time.Millisecond)

	// list the job history
	historyResponse, err := s.client.Jobs().History(ctx, &apimodels.ListJobHistoryRequest{JobIDOrName: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(historyResponse)
	s.Require().NotEmpty(historyResponse.Items)
	for _, h := range historyResponse.Items {
		s.Require().Equal(putResponse.JobID, h.JobID)
	}

	// list executions
	executionsResponse, err := s.client.Jobs().Executions(ctx, &apimodels.ListJobExecutionsRequest{JobIDOrName: putResponse.
		JobID})
	s.Require().NoError(err)
	s.Require().NotNil(executionsResponse)
	s.Require().NotEmpty(executionsResponse.Items)

	// list results
	resultsResponse, err := s.client.Jobs().Results(ctx, &apimodels.ListJobResultsRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(resultsResponse)
	s.Require().Empty(resultsResponse.Items, "expected no results as we did not specify a publisher")

	// stop the job should fail as it is already complete
	_, err = s.client.Jobs().Stop(ctx, &apimodels.StopJobRequest{JobID: putResponse.JobID})
	s.Require().Error(err)
}
