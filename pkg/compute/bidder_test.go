//go:build unit || !integration

package compute_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
)

type BidderSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	mockSemanticStrategy *bidstrategy.MockSemanticBidStrategy
	mockResourceStrategy *bidstrategy.MockResourceBidStrategy
	mockExecutionStore   store.ExecutionStore
	bidder               compute.Bidder
}

func TestBidderSuite(t *testing.T) {
	suite.Run(t, new(BidderSuite))
}

func (s *BidderSuite) SetupTest() {
	ctx := context.Background()
	execStore, err := boltdb.NewStore(ctx,
		filepath.Join(s.T().TempDir(), "bidder-test.db"))
	s.Require().NoError(err)
	s.T().Cleanup(func() {
		execStore.Close(ctx)
	})

	s.ctrl = gomock.NewController(s.T())
	s.mockSemanticStrategy = bidstrategy.NewMockSemanticBidStrategy(s.ctrl)
	s.mockResourceStrategy = bidstrategy.NewMockResourceBidStrategy(s.ctrl)
	s.mockExecutionStore = execStore
	s.bidder = compute.NewBidder(compute.BidderParams{
		SemanticStrategy: []bidstrategy.SemanticBidStrategy{s.mockSemanticStrategy},
		ResourceStrategy: []bidstrategy.ResourceBidStrategy{s.mockResourceStrategy},
		Store:            s.mockExecutionStore,
		UsageCalculator:  capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{Defaults: models.Resources{}}),
	})
}

func (s *BidderSuite) TestRunBidding() {
	ctx := context.Background()
	job := mock.Job()
	execution := mock.ExecutionForJob(job)

	tests := []struct {
		name                   string
		expectedExecutionState models.ExecutionStateType
		mockExpectations       func()
	}{
		{
			name: "semantic should not bid",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false}, nil)
			},
			expectedExecutionState: models.ExecutionStateAskForBidRejected,
		},
		{
			name: "semantic and resource should bid",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)
			},
			expectedExecutionState: models.ExecutionStateAskForBidAccepted,
		},
		{
			name: "semantic should bid but resource should not",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false}, nil)
			},
			expectedExecutionState: models.ExecutionStateAskForBidRejected,
		},
		{
			name: "semantic bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("semantic error"))
			},
			expectedExecutionState: models.ExecutionStateFailed,
		},
		{
			name: "resource bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("resource error"))
			},
			expectedExecutionState: models.ExecutionStateFailed,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			tt.mockExpectations()
			err := s.mockExecutionStore.CreateExecution(ctx, *execution)
			s.Require().NoError(err)

			s.bidder.RunBidding(ctx, execution)

			updatedExecution, err := s.mockExecutionStore.GetExecution(ctx, execution.ID)
			s.Require().NoError(err)
			s.Equal(tt.expectedExecutionState, updatedExecution.ComputeState.StateType,
				"expected execution state %s but got %s", tt.expectedExecutionState, updatedExecution.ComputeState.StateType)
		})
	}
}

func (s *BidderSuite) TestRunBidding_PendingApproval() {
	ctx := context.Background()
	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	execution.DesiredState = models.NewExecutionDesiredState(models.ExecutionDesiredStatePending)

	s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
		Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)
	s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
		Return(bidstrategy.BidStrategyResponse{ShouldBid: true}, nil)

	err := s.mockExecutionStore.CreateExecution(ctx, *execution)
	s.Require().NoError(err)

	s.bidder.RunBidding(ctx, execution)

	updatedExecution, err := s.mockExecutionStore.GetExecution(ctx, execution.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecutionStateAskForBidAccepted, updatedExecution.ComputeState.StateType)
}
