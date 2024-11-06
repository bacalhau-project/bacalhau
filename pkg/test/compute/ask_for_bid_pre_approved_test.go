//go:build integration || !unit

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type AskForBidPreApprovedSuite struct {
	ComputeSuite
}

func TestAskForBidPreApprovedSuite(t *testing.T) {
	suite.Run(t, new(AskForBidPreApprovedSuite))
}

func (s *AskForBidPreApprovedSuite) TestAskForBid() {
	s.runAskForBidTest(bidResponseTestCase{})
}

func (s *AskForBidPreApprovedSuite) verify(executionID string, expected models.Resources) {
	execution, err := s.node.ExecutionStore.GetExecution(context.Background(), executionID)
	s.NoError(err)
	s.Equal(expected, *execution.TotalAllocatedResources())
}

func (s *AskForBidPreApprovedSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{})
	s.verify(response, s.config.SystemConfig.DefaultComputeJobResourceLimits)
}

func (s *AskForBidPreApprovedSuite) TestUseSubmittedResourceUsage() {
	usage := models.Resources{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageBelowLimits() {
	jobResources := s.capacity
	jobResources.CPU = s.capacity.CPU / 2
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), jobResources),
	})
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), s.capacity),
	})
}

func (s *AskForBidPreApprovedSuite) TestRejectUsageExceedingLimits() {
	jobResources := s.capacity
	jobResources.CPU += 0.01
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), jobResources),
		rejected:  true,
	})
}

func (s *AskForBidPreApprovedSuite) runAskForBidTest(testCase bidResponseTestCase) string {
	ctx := context.Background()

	// setup default values
	execution := testCase.execution
	if execution == nil {
		execution = mock.Execution()
	}
	execution.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)

	_, err := s.node.LocalEndpoint.AskForBid(ctx, legacy.AskForBidRequest{
		RoutingMetadata: legacy.RoutingMetadata{
			TargetPeerID: s.node.ID,
			SourcePeerID: s.node.ID,
		},
		Execution:       execution,
		WaitForApproval: false,
	})
	s.NoError(err)

	// Always expect a bid response
	select {
	case bid := <-s.bidChannel:
		s.Equal(!testCase.rejected, bid.Accepted, "unexpected bid acceptance state")
	case <-s.failureChannel:
		s.Fail("Got unexpected failure")
	case <-time.After(5 * time.Second):
		s.Fail("Timeout waiting for bid")
	}

	// For accepted bids, also expect a completion
	if !testCase.rejected {
		select {
		case <-s.completedChannel:
			s.T().Log("Received expected completion")
		case <-s.failureChannel:
			s.Fail("Got unexpected failure")
		case <-s.bidChannel:
			s.Fail("Got unexpected second bid")
		case <-time.After(5 * time.Second):
			s.Fail("Timeout waiting for completion")
		}
	}

	// Verify final execution state
	retrievedExecution, err := s.node.ExecutionStore.GetExecution(ctx, execution.ID)
	s.Require().NoError(err)

	expectedState := models.ExecutionStateCompleted
	if testCase.rejected {
		expectedState = models.ExecutionStateAskForBidRejected
	}
	s.Equal(expectedState, retrievedExecution.ComputeState.StateType,
		"expected execution state %s but got %s", expectedState, retrievedExecution.ComputeState.StateType)

	return execution.ID
}
