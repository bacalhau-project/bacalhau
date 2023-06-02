//go:build integration || !unit

package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type APITestSuite struct {
	DashboardTestSuite
}

var _ suite.SetupAllSuite = (*APITestSuite)(nil)
var _ suite.SetupTestSuite = (*APITestSuite)(nil)
var _ suite.TearDownTestSuite = (*APITestSuite)(nil)

func TestDashboardAPIs(t *testing.T) {
	suite.Run(t, new(APITestSuite))
}

func (s *APITestSuite) TestJobsCountInitiallyZero() {
	count, err := s.api.GetJobsCount(s.ctx, localdb.JobQuery{ReturnAll: true})
	s.NoError(err)
	s.Equal(0, count)
}

func (s *APITestSuite) TestCanaryJobsStored() {
	s.T().Skip("unsupported")
	/*
		jobEvent := v1beta1.JobEvent{
			JobID:     "testjob",
			EventName: v1beta1.JobEventCreated,
			Spec: v1beta1.Spec{
				Annotations: []string{"canary"},
			},
		}
		s.Require().NoError(s.api.AddEvent(jobEvent))

		info, err := s.api.GetJobInfo(s.ctx, jobEvent.JobID)
		s.Require().NoError(err)
		s.Require().Contains(info.Job.Spec.Annotations, "canary")

	*/
}

func (s *APITestSuite) TestHelloLambdaJobsStored() {
	s.T().Skip("unsupported")
	/*
		jobEvent := v1beta1.JobEvent{
			JobID:     "testjob",
			EventName: v1beta1.JobEventCreated,
			Spec: v1beta1.Spec{
				Docker: v1beta1.JobSpecDocker{
					Entrypoint: []string{"hello λ!"},
				},
			},
		}
		s.Require().NoError(s.api.AddEvent(jobEvent))

		info, err := s.api.GetJobInfo(s.ctx, jobEvent.JobID)
		s.Require().NoError(err)
		s.Require().Contains(info.Job.Spec.Docker.Entrypoint, "hello λ!")

	*/
}

func (s *APITestSuite) TestModerateBidRequest() {
	for _, shouldApprove := range []bool{true, false} {
		s.Run(fmt.Sprintf("moderator approves is %t", shouldApprove), func() {
			job := v1beta1.Job{Metadata: v1beta1.Metadata{ID: s.T().Name()}}
			s.NoError(s.localDB.AddJob(s.ctx, &job))

			resp, err := s.api.ShouldExecuteJob(s.ctx, &bidstrategy.JobSelectionPolicyProbeData{
				JobID: job.Metadata.ID,
			})
			s.NoError(err)
			s.NotNil(resp)
			s.Equal(true, resp.ShouldWait)

			info, err := s.api.GetJobInfo(s.ctx, job.Metadata.ID)
			s.Require().NoError(err)
			s.Require().Equal(1, len(info.Requests))
			request := info.Requests[0]

			err = s.api.ModerateJob(s.ctx, request.GetID(), "looks great", shouldApprove, s.user)
			s.NoError(err)

			resp, err = s.api.ShouldExecuteJob(s.ctx, &bidstrategy.JobSelectionPolicyProbeData{JobID: job.Metadata.ID})
			s.NoError(err)
			s.Equal(shouldApprove, resp.ShouldBid)
			s.Equal(false, resp.ShouldWait)
			s.Equal("looks great", resp.Reason)
		})
	}
}

func (s *APITestSuite) TestModerationTriggersCallback() {
	job := v1beta1.Job{Metadata: v1beta1.Metadata{ID: s.T().Name()}}
	s.NoError(s.localDB.AddJob(s.ctx, &job))

	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		req, err := publicapi.UnmarshalSigned[bidstrategy.ModerateJobRequest](r.Context(), r.Body)
		s.NoError(err)

		s.Equal(job.Metadata.ID, req.JobID)
		s.Equal(system.GetClientID(), req.ClientID)

		resp := req.Response
		s.Equal(true, resp.ShouldBid)
		s.Equal(false, resp.ShouldWait)
		s.Equal("looks great", resp.Reason)
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	url, err := url.Parse(server.URL)
	s.NoError(err)

	resp, err := s.api.ShouldExecuteJob(s.ctx, &bidstrategy.JobSelectionPolicyProbeData{
		JobID:    job.Metadata.ID,
		Callback: url,
	})
	s.NoError(err)
	s.Equal(true, resp.ShouldWait)

	info, err := s.api.GetJobInfo(s.ctx, job.Metadata.ID)
	s.Require().NoError(err)
	s.Require().Equal(1, len(info.Requests))
	request := info.Requests[0]

	err = s.api.ModerateJob(s.ctx, request.GetID(), "looks great", true, s.user)
	s.NoError(err)
	s.Equal(true, called)
}
