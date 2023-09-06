//go:build unit || !integration

package test

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func (s *ServerSuite) TestJobOperations() {
	job := mock.Job()
	putResponse, err := s.client.Jobs().Put(&apimodels.PutJobRequest{Job: job})
	s.Require().NoError(err)
	s.Require().NotNil(putResponse)
	s.Require().NotEmpty(putResponse.JobID)
	s.Require().NotEmpty(putResponse.EvaluationID)

	// retrieve the job
	getResponse, err := s.client.Jobs().Get(&apimodels.GetJobRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(getResponse)
	s.Require().Equal(putResponse.JobID, getResponse.Job.ID)
	s.Require().EqualValues(job.Tasks, getResponse.Job.Tasks)

	// list the job
	listResponse, err := s.client.Jobs().List(&apimodels.ListJobsRequest{})
	s.Require().NoError(err)
	s.Require().NotNil(listResponse)
	s.Require().NotEmpty(listResponse.Jobs)

	found := false
	for _, j := range listResponse.Jobs {
		if j.ID == putResponse.JobID {
			found = true
			break
		}
	}
	s.Require().True(found, "job %s not found in list", putResponse.JobID)

	// Wait for job executions to start, and for the job to complete
	s.Eventually(func() bool {
		res, err := s.client.Jobs().Get(&apimodels.GetJobRequest{JobID: putResponse.JobID})
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
	historyResponse, err := s.client.Jobs().History(&apimodels.ListJobHistoryRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(historyResponse)
	s.Require().NotEmpty(historyResponse.History)
	for _, h := range historyResponse.History {
		s.Require().Equal(putResponse.JobID, h.JobID)
	}

	// list executions
	executionsResponse, err := s.client.Jobs().Executions(&apimodels.ListJobExecutionsRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(executionsResponse)
	s.Require().NotEmpty(executionsResponse.Executions)

	// list results
	resultsResponse, err := s.client.Jobs().Results(&apimodels.ListJobResultsRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(resultsResponse)
	s.Require().NotEmpty(resultsResponse.Results)

	// stop the job should fail as it is already complete
	_, err = s.client.Jobs().Stop(&apimodels.StopJobRequest{JobID: putResponse.JobID})
	s.Require().Error(err)
}
