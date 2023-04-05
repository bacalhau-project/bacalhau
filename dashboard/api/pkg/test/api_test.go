//go:build integration || !unit

package test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	dashtypes "github.com/bacalhau-project/bacalhau/dashboard/api/pkg/types"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/localdb"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/suite"
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

func (s *APITestSuite) TestModerateJobWithoutRequest() {
	job := v1beta1.Job{Metadata: v1beta1.Metadata{ID: "testjob"}}
	s.NoError(s.localDB.AddJob(s.ctx, &job))

	err := s.api.ModerateJobWithoutRequest(s.ctx, job.Metadata.ID, "looks great", true, dashtypes.ModerationTypeDatacap, s.user)
	s.NoError(err)

	info, err := s.api.GetJobInfo(s.ctx, job.Metadata.ID)
	s.NoError(err)
	s.Equal(1, len(info.Moderations))

	summary := info.Moderations[0]
	s.NotNil(summary.User)
	s.Equal(s.user.Username, summary.User.Username)
	s.Equal(true, summary.Moderation.Status)
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

			err = s.api.ModerateJobWithoutRequest(
				s.ctx,
				job.Metadata.ID,
				"looks great",
				shouldApprove,
				dashtypes.ModerationTypeExecution,
				s.user,
			)
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

	err = s.api.ModerateJobWithoutRequest(
		s.ctx,
		job.Metadata.ID,
		"looks great",
		true,
		dashtypes.ModerationTypeExecution,
		s.user,
	)
	s.NoError(err)
	s.Equal(true, called)
}
