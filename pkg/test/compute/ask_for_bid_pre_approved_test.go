//go:build integration || !unit

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
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
	localState, err := s.node.ExecutionStore.GetExecution(context.Background(), executionID)
	s.NoError(err)
	s.Equal(expected, *localState.Execution.TotalAllocatedResources())
}

func (s *AskForBidPreApprovedSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{})
	s.verify(response, s.config.DefaultJobResourceLimits)
}

func (s *AskForBidPreApprovedSuite) TestUseSubmittedResourceUsage() {
	usage := models.Resources{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageBelowLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU / 2}),
	})
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU}),
	})
}

func (s *AskForBidPreApprovedSuite) TestRejectUsageExceedingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		execution: addResourceUsage(mock.Execution(),
			models.Resources{CPU: s.config.JobResourceLimits.CPU + 0.01}),
		rejected: true,
	})
}

func (s *AskForBidPreApprovedSuite) runAskForBidTest(testCase bidResponseTestCase) string {
	ctx := context.Background()

	// setup default values
	execution := testCase.execution
	if execution == nil {
		execution = mock.Execution()
	}
	_, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		RoutingMetadata: compute.RoutingMetadata{
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

	localExecutionState, err := s.node.ExecutionStore.GetExecution(ctx, execution.ID)
	if testCase.rejected {
		s.ErrorIs(err, store.NewErrExecutionNotFound(execution.ID))
	} else {
		s.NoError(err)
		s.Equal(store.ExecutionStateCompleted, localExecutionState.State)
	}
	return execution.ID
}
