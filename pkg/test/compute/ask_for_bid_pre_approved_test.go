//go:build integration || !unit

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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

func (s *AskForBidPreApprovedSuite) verify(executionID string, expected model.ResourceUsageData) {
	execution, err := s.node.ExecutionStore.GetExecution(context.Background(), executionID)
	s.NoError(err)
	s.Equal(expected, execution.ResourceUsage)
}

func (s *AskForBidPreApprovedSuite) TestPopulateResourceUsage() {
	response := s.runAskForBidTest(bidResponseTestCase{})
	s.verify(response, s.config.DefaultJobResourceLimits)
}

func (s *AskForBidPreApprovedSuite) TestUseSubmittedResourceUsage() {
	usage := model.ResourceUsageData{CPU: 1, Memory: 2, Disk: 3}
	response := s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(), usage),
	})
	s.verify(response, usage)
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageBelowLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU / 2}),
	})
}

func (s *AskForBidPreApprovedSuite) TestAcceptUsageMatachingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU}),
	})
}

func (s *AskForBidPreApprovedSuite) TestRejectUsageExceedingLimits() {
	s.runAskForBidTest(bidResponseTestCase{
		job: addResourceUsage(generateJob(),
			model.ResourceUsageData{CPU: s.config.JobResourceLimits.CPU + 0.01}),
		rejected: true,
	})
}

func (s *AskForBidPreApprovedSuite) runAskForBidTest(testCase bidResponseTestCase) string {
	ctx := context.Background()

	// setup default values
	job := testCase.job
	job.Metadata.Requester.RequesterNodeID = s.node.ID
	if job.Metadata.ID == "" {
		job = generateJob()
	}

	executionID := uuid.NewString()
	_, err := s.node.LocalEndpoint.AskForBid(ctx, compute.AskForBidRequest{
		ExecutionMetadata: compute.ExecutionMetadata{
			JobID:       job.Metadata.ID,
			ExecutionID: executionID,
		},
		RoutingMetadata: compute.RoutingMetadata{
			TargetPeerID: s.node.ID,
			SourcePeerID: s.node.ID,
		},
		Job:             job,
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

	execution, err := s.node.ExecutionStore.GetExecution(ctx, executionID)
	if testCase.rejected {
		s.ErrorIs(err, store.NewErrExecutionNotFound(executionID))
	} else {
		s.NoError(err)
		s.Equal(store.ExecutionStateCompleted, execution.State)
	}
	return executionID
}
