//go:build integration || !unit

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
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
	_, err := s.node.LocalEndpoint.AskForBid(ctx, messages.AskForBidRequest{
		RoutingMetadata: messages.RoutingMetadata{
			TargetPeerID: s.node.ID,
			SourcePeerID: s.node.ID,
		},
		Execution:       execution,
		WaitForApproval: false,
	})
	s.NoError(err)

	select {
	case result := <-s.completedChannel:
		s.False(testCase.rejected, "unexpected completion: %v", result)
	case failure := <-s.failureChannel:
		s.True(testCase.rejected, "unexpected failure: %v", failure)
	case bid := <-s.bidChannel:
		s.Fail("unexpected bid: %v", bid)
	case <-time.After(5 * time.Second):
		s.Fail("did not receive bid, completion or failure")
	}

	retrievedExecution, err := s.node.ExecutionStore.GetExecution(ctx, execution.ID)
	if testCase.rejected {
		s.ErrorIs(err, store.NewErrExecutionNotFound(execution.ID))
	} else {
		s.NoError(err)
		s.Equal(models.ExecutionStateCompleted, retrievedExecution.ComputeState.StateType)
	}
	return execution.ID
}
