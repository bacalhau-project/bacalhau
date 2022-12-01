package compute

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/compute/frontend"
	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	"github.com/filecoin-project/bacalhau/pkg/compute/store/resolver"
	"github.com/google/uuid"
)

func (s *ComputeSuite) TestBidRejected() {
	ctx := context.Background()
	executionID := s.prepareAndAskForBid(ctx, generateJob())

	_, err := s.node.Frontend.BidRejected(ctx, frontend.BidRejectedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCancelled))
	s.NoError(err)
}

func (s *ComputeSuite) TestBidRejected_DoesntExist() {
	ctx := context.Background()
	_, err := s.node.Frontend.BidRejected(ctx, frontend.BidRejectedRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *ComputeSuite) TestBidRejected_WrongState() {
	ctx := context.Background()

	// loop over few states to make sure we don't accept bids, if state is not `Created`
	for _, state := range []store.ExecutionState{
		store.ExecutionStateWaitingVerification,
		store.ExecutionStatePublishing,
		store.ExecutionStateCancelled,
		store.ExecutionStateCompleted,
	} {
		executionID := s.prepareAndAskForBid(ctx, generateJob())
		err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: executionID,
			NewState:    state,
		})
		s.NoError(err)

		_, err = s.node.Frontend.BidRejected(ctx, frontend.BidRejectedRequest{ExecutionID: executionID})
		s.Error(err)
	}
}
