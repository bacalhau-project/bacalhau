//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/resolver"
)

type BidAcceptedSuite struct {
	ComputeSuite
}

func TestBidAcceptedSuite(t *testing.T) {
	suite.Run(t, new(BidAcceptedSuite))
}

func (s *BidAcceptedSuite) TestBidAccepted() {
	ctx := context.Background()
	executionID := s.prepareAndAskForBid(ctx, mock.Execution())

	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(store.ExecutionStateCompleted))
	s.NoError(err)
}

func (s *BidAcceptedSuite) TestDoesntExist() {
	ctx := context.Background()
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *BidAcceptedSuite) TestWrongState() {
	ctx := context.Background()

	// loop over few states to make sure we don't accept bids, if state is not `Created`
	for _, state := range []store.LocalStateType{
		store.ExecutionStatePublishing,
		store.ExecutionStateCancelled,
		store.ExecutionStateCompleted,
	} {
		executionID := s.prepareAndAskForBid(ctx, mock.Execution())
		err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionStateRequest{
			ExecutionID: executionID,
			NewState:    state,
		})
		s.NoError(err)

		_, err = s.node.LocalEndpoint.BidAccepted(ctx, compute.BidAcceptedRequest{ExecutionID: executionID})
		s.Error(err)
	}
}
