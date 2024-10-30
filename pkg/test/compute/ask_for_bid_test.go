//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
)

type AskForBidSuite struct {
	ComputeSuite
}

type bidResponseTestCase struct {
	name      string
	execution *models.Execution
	rejected  bool
}

func TestAskForBidSuite(t *testing.T) {
	suite.Run(t, new(AskForBidSuite))
}

func (s *AskForBidSuite) TestAskForBid() {
	s.runAskForBidTest(bidResponseTestCase{name: "empty test case"})
}

func (s *AskForBidSuite) verify(response compute.BidResult, expected models.Resources) {
	execution, err := s.node.ExecutionStore.GetExecution(context.Background(), response.ExecutionID)
	s.NoError(err)
	s.Equal(expected, *execution.TotalAllocatedResources())
}

func (s *AskForBidSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{name: "populate resrouce usage"})
	s.verify(response, s.config.SystemConfig.DefaultComputeJobResourceLimits)
}

func (s *AskForBidSuite) TestUseSubmittedResourceUsage() {
	usage := models.Resources{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		name:      "use submitted resource usage",
		execution: addResourceUsage(mock.Execution(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidSuite) TestAcceptUsageBelowLimits() {
	jobResources := s.capacity
	jobResources.CPU = s.capacity.CPU / 2
	s.runAskForBidTest(bidResponseTestCase{
		name:      "accept usage below limits",
		execution: addResourceUsage(mock.Execution(), jobResources),
	})
}

func (s *AskForBidSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		name:      "accept usage matching limits",
		execution: addResourceUsage(mock.Execution(), s.capacity),
	})
}

func (s *AskForBidSuite) TestRejectUsageExceedingLimits() {
	jobResources := s.capacity
	jobResources.CPU += 0.01
	s.runAskForBidTest(bidResponseTestCase{
		name:      "reject usage exceeding limits",
		execution: addResourceUsage(mock.Execution(), jobResources),
		rejected:  true,
	})
}

func (s *AskForBidSuite) runAskForBidTest(testCase bidResponseTestCase) compute.BidResult {
	ctx := context.Background()

	var result compute.BidResult
	_ = s.Run(testCase.name, func() {
		// setup default values
		execution := testCase.execution
		if execution == nil {
			execution = mock.Execution()
		}

		result = s.askForBid(ctx, execution)
		s.Equal(!testCase.rejected, result.Accepted)

		// check execution state
		execution, err := s.node.ExecutionStore.GetExecution(ctx, result.ExecutionID)
		if testCase.rejected {
			s.ErrorIs(err, store.NewErrExecutionNotFound(result.ExecutionID))
		} else {
			s.NoError(err)
			s.Equal(models.ExecutionStateNew, execution.ComputeState.StateType)
		}
	})

	return result
}
