//go:build integration || !unit

package test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/stretchr/testify/suite"
)

type AutoSelectionSuite struct {
	DashboardTestSuite
}

func TestAutoSelectionSuite(t *testing.T) {
	suite.Run(t, new(AutoSelectionSuite))
}

func (s *AutoSelectionSuite) SetupTest() {
	// Don't hold test jobs for moderation for these tests.
	s.opts.SelectionPolicy.RejectStatelessJobs = false
	s.DashboardTestSuite.SetupTest()
}

func (s *AutoSelectionSuite) TestApprovesImmediately() {
	job := v1beta1.Job{Metadata: v1beta1.Metadata{ID: s.T().Name()}}
	s.NoError(s.localDB.AddJob(s.ctx, &job))

	resp, err := s.api.ShouldExecuteJob(s.ctx, &bidstrategy.JobSelectionPolicyProbeData{
		JobID: job.Metadata.ID,
	})
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(true, resp.ShouldBid)
	s.Equal(false, resp.ShouldWait)
}
