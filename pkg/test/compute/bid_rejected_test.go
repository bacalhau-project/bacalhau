//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
)

type BidRejectedSuite struct {
	ComputeSuite
}

func TestBidRejectedSuite(t *testing.T) {
	suite.Run(t, new(BidRejectedSuite))
}

func (s *BidRejectedSuite) TestBidRejected() {
	ctx := context.Background()
	executionID := s.prepareAndAskForBid(ctx, generateJob(s.T()))

	_, err := s.node.LocalEndpoint.BidRejected(ctx, compute.BidRejectedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCancelled))
	s.NoError(err)
}

func (s *BidRejectedSuite) TestDoesntExist() {
	ctx := context.Background()
	_, err := s.node.LocalEndpoint.BidRejected(ctx, compute.BidRejectedRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *BidRejectedSuite) TestWrongState() {
	ctx := context.Background()

	// loop over few states to make sure we don't accept bids, if state is not `Created`
	for _, state := range []store.LocalStateType{
		store.ExecutionStatePublishing,
		store.ExecutionStateCancelled,
		store.ExecutionStateCompleted,
	} {
		executionID := s.prepareAndAskForBid(ctx, generateJob(s.T()))
		err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: executionID,
			NewState:    state,
		})
		s.NoError(err)

		_, err = s.node.LocalEndpoint.BidRejected(ctx, compute.BidRejectedRequest{ExecutionID: executionID})
		s.Error(err)
	}
}
