//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type AskForBidSuite struct {
	ComputeSuite
}

type bidResponseTestCase struct {
	name          string
	job           model.Job
	rejected      bool
	resourceUsage model.ResourceUsageData
}

func TestAskForBidSuite(t *testing.T) {
	suite.Run(t, new(AskForBidSuite))
}

func (s *AskForBidSuite) TestAskForBid() {
	s.runAskForBidTest(bidResponseTestCase{})
}

func (s *AskForBidSuite) verify(response compute.BidResult, expected model.ResourceUsageData) {
	execution, err := s.node.ExecutionStore.GetExecution(context.Background(), response.ExecutionID)
	s.NoError(err)
	s.Equal(expected, execution.ResourceUsage)
}

func (s *AskForBidSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{})
	s.verify(response, s.config.DefaultJobResourceLimits)
}

func (s *AskForBidSuite) TestUseSubmittedResourceUsage() {
	usage := model.ResourceUsageData{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidSuite) TestAcceptUsageBelowLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU / 2}),
	})
}

func (s *AskForBidSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU}),
	})
}

func (s *AskForBidSuite) TestRejectUsageExceedingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU + 0.01}),
		rejected: true,
	})
}

func (s *AskForBidSuite) runAskForBidTest(testCase bidResponseTestCase) compute.BidResult {
	ctx := context.Background()

	// setup default values
	job := testCase.job
	job.Metadata.Requester.RequesterNodeID = s.node.ID
	if job.Metadata.ID == "" {
		job = generateJob()
	}

	result := s.askForBid(ctx, job)
	s.Equal(!testCase.rejected, result.Accepted)

	// check execution state
	execution, err := s.node.ExecutionStore.GetExecution(ctx, result.ExecutionID)
	if testCase.rejected {
		s.ErrorIs(err, store.NewErrExecutionNotFound(result.ExecutionID))
	} else {
		s.NoError(err)
		s.Equal(store.ExecutionStateCreated, execution.State)
	}

	return result
}
