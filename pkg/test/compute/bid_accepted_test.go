//go:build integration || !unit

package compute

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

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

	_, err := s.node.LocalEndpoint.BidAccepted(ctx, messages.BidAcceptedRequest{ExecutionID: executionID})
	s.NoError(err)
	err = s.stateResolver.Wait(ctx, executionID, resolver.CheckForState(models.ExecutionStateCompleted))
	s.NoError(err)
}

func (s *BidAcceptedSuite) TestDoesntExist() {
	ctx := context.Background()
	_, err := s.node.LocalEndpoint.BidAccepted(ctx, messages.BidAcceptedRequest{ExecutionID: uuid.NewString()})
	s.Error(err)
}

func (s *BidAcceptedSuite) TestWrongState() {
	// loop over few states to make sure we don't accept bids, if state is not `Created`
	for _, state := range []models.ExecutionStateType{
		models.ExecutionStatePublishing,
		models.ExecutionStateCancelled,
		models.ExecutionStateCompleted,
	} {
		s.Run(state.String(), func() {
			ctx := context.Background()

			executionID := s.prepareAndAskForBid(ctx, mock.Execution())
			err := s.node.ExecutionStore.UpdateExecutionState(ctx, store.UpdateExecutionRequest{
				ExecutionID: executionID,
				NewValues:   models.Execution{ComputeState: models.NewExecutionState(state)},
			})
			s.NoError(err)

			_, err = s.node.LocalEndpoint.BidAccepted(ctx, messages.BidAcceptedRequest{ExecutionID: executionID})
			s.Error(err)
		})
	}
}
