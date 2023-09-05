//go:build unit || !integration

package test

import (
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func (s *ServerSuite) TestPutJob() {
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

	// list the job history
	historyResponse, err := s.client.Jobs().History(&apimodels.ListJobHistoryRequest{JobID: putResponse.JobID})
	s.Require().NoError(err)
	s.Require().NotNil(historyResponse)
	s.Require().NotEmpty(historyResponse.History)
	for _, h := range historyResponse.History {
		s.Require().Equal(putResponse.JobID, h.JobID)
	}
}

//
//func (s *ServerSuite) TestGetJob() {
//	_, err := s.client.Jobs().Get(&apimodels.GetJobRequest{JobID: "test-job-id"})
//	s.Require().NoError(err)
//
//}
//
//func (s *ServerSuite) TestListJobs() {
//	_, err := s.client.Jobs().List(&apimodels.ListJobsRequest{})
//	s.Require().NoError(err)
//}
//
//func (s *ServerSuite) TestStopJob() {
//	_, err := s.client.Jobs().Stop(&apimodels.StopJobRequest{})
//	s.Require().NoError(err)
//}
//
//func (s *ServerSuite) TestSummarizeJob() {
//	_, err := s.client.Jobs().Summarize(&apimodels.SummarizeJobRequest{})
//	s.Require().NoError(err)
//}
//
//func (s *ServerSuite) TestDescribeJob() {
//	_, err := s.client.Jobs().Describe(&apimodels.DescribeJobRequest{})
//	s.Require().NoError(err)
//}
