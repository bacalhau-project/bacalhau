//go:build integration

package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type bidResponseTestCase struct {
	name          string
	job           model.Job
	rejected      bool
	resourceUsage model.ResourceUsageData
}

func (s *ComputeSuite) TestAskForBid() {
	s.runAskForBidTest(bidResponseTestCase{})
}

func (s *ComputeSuite) TestAskForBid_PopulateResourceUsage() {
	ctx := context.Background()
	verify := func(response compute.AskForBidResponse, expected model.ResourceUsageData) {
		execution, err := s.node.ExecutionStore.GetExecution(ctx, response.ExecutionID)
		s.NoError(err)
		s.Equal(expected, execution.ResourceUsage)
	}

	s.Run("populate default usage", func() {
		response := s.runAskForBidTest(bidResponseTestCase{})
		verify(response, s.config.DefaultJobResourceLimits)
	})

	s.Run("use submitted usage", func() {
		usage := model.ResourceUsageData{CPU: 1, Memory: 2, Disk: 3}
		response := s.runAskForBidTest(bidResponseTestCase{
			job: addResourceUsage(generateJob(), usage),
		})
		verify(response, usage)
	})
}

func (s *ComputeSuite) TestAskForBid_JobResourceLimits() {
	s.Run("accept usage below limits", func() {
		s.runAskForBidTest(bidResponseTestCase{
			job: addResourceUsage(generateJob(),
				model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU / 2}),
		})
	})

	s.Run("accept usage matching limits", func() {
		s.runAskForBidTest(bidResponseTestCase{
			job: addResourceUsage(generateJob(),
				model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU}),
		})
	})

	s.Run("reject usage exceeding limits", func() {
		s.runAskForBidTest(bidResponseTestCase{
			job: addResourceUsage(generateJob(),
				model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU + 0.01}),
			rejected: true,
		})
	})

}

func (s *ComputeSuite) TestAskForBid_RejectStateless() {
	s.config.JobSelectionPolicy.RejectStatelessJobs = true
	s.setupNode()

	s.Run("reject stateless", func() {
		s.runAskForBidTest(bidResponseTestCase{
			rejected: true,
		})
	})

	s.Run("accept stateful", func() {
		s.runAskForBidTest(bidResponseTestCase{
			job: addInput(generateJob(), "cid"),
		})
	})
}

func (s *ComputeSuite) runAskForBidTest(testCase bidResponseTestCase) compute.AskForBidResponse {
	ctx := context.Background()

	// setup default values
	job := testCase.job
	if job.Metadata.ID == "" {
		job = generateJob()
	}

	// issue the request
	request := compute.AskForBidRequest{
		Job: job,
	}
	response, err := s.node.LocalEndpoint.AskForBid(ctx, request)
	s.NoError(err)

	// check the response
	s.Equal(!testCase.rejected, response.Accepted)

	// check execution state
	if !testCase.rejected {
		execution, err := s.node.ExecutionStore.GetExecution(ctx, response.ExecutionID)
		s.NoError(err)
		s.Equal(store.ExecutionStateCreated, execution.State)
	}

	return response
}
