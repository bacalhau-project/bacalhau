//go:build integration || !unit

package test

import (
	"encoding/json"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta1"
	"github.com/bacalhau-project/bacalhau/pkg/verifier"
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
	_, err := s.localDB.GetJob(s.ctx, job.Metadata.ID)
	s.Require().NoError(err)

	resp, err := s.api.ShouldExecuteJob(s.ctx, &bidstrategy.JobSelectionPolicyProbeData{
		JobID: job.Metadata.ID,
	})
	s.NoError(err)
	s.NotNil(resp)
	s.Equal(true, resp.ShouldBid)
	s.Equal(false, resp.ShouldWait)
}

func (s *AutoSelectionSuite) TestValidatesImmediately() {
	job := v1beta1.Job{Metadata: v1beta1.Metadata{ID: s.T().Name()}}
	s.Require().NoError(s.localDB.AddJob(s.ctx, &job))
	s.Require().Empty(job.Spec.Inputs)

	storageSpec := model.StorageSpec{StorageSource: model.StorageSourceIPFS}
	specBytes, err := json.Marshal(&storageSpec)
	s.Require().NoError(err)

	execution := model.ExecutionState{
		JobID:                job.Metadata.ID,
		State:                model.ExecutionStateResultProposed,
		VerificationProposal: specBytes,
	}

	resp, err := s.api.ShouldVerifyJob(s.ctx, verifier.VerifierRequest{
		JobID:      job.Metadata.ID,
		Executions: []model.ExecutionState{execution},
	})
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)
	s.Require().Equal(execution.ID(), resp[0].ExecutionID)
	s.Require().True(resp[0].Verified)
}
