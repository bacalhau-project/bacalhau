//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type AskForBidSuite struct {
	ComputeSuite
}

type bidResponseTestCase struct {
	name          string
	execution     *models.Execution
	rejected      bool
	resourceUsage models.Resources
}

func TestAskForBidSuite(t *testing.T) {
	suite.Run(t, new(AskForBidSuite))
}

func (s *AskForBidSuite) TestAskForBid() {
	s.runAskForBidTest(bidResponseTestCase{})
}

func (s *AskForBidSuite) verify(response compute.BidResult, expected models.Resources) {
	localState, err := s.node.ExecutionStore.GetExecution(context.Background(), response.ExecutionID)
	s.NoError(err)
	s.Equal(expected, *localState.Execution.TotalAllocatedResources())
}

func (s *AskForBidSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{})
	s.verify(response, s.config.DefaultJobResourceLimits)
}

func (s *AskForBidSuite) TestUseSubmittedResourceUsage() {
	usage := models.Resources{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidSuite) TestAcceptUsageBelowLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU / 2}),
	})
}

func (s *AskForBidSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU}),
	})
}

func (s *AskForBidSuite) TestRejectUsageExceedingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU + 0.01}),
		rejected: true,
	})
}

func (s *AskForBidSuite) runAskForBidTest(testCase bidResponseTestCase) compute.BidResult {
	ctx := context.Background()

	// setup default values
	execution := testCase.execution
	if execution == nil {
		execution = mock.Execution()
	}

	result := s.askForBid(ctx, execution)
	s.Equal(!testCase.rejected, result.Accepted)

	// check execution state
	localExecutionState, err := s.node.ExecutionStore.GetExecution(ctx, result.ExecutionID)
	if testCase.rejected {
		s.ErrorIs(err, store.NewErrExecutionNotFound(result.ExecutionID))
	} else {
		s.NoError(err)
		s.Equal(store.ExecutionStateCreated, localExecutionState.State)
	}

	return result
}
