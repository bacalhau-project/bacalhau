//go:build unit || !integration

package compute_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
)

type BidderSuite struct {
	suite.Suite
	ctrl                 *gomock.Controller
	mockSemanticStrategy *bidstrategy.MockSemanticBidStrategy
	mockResourceStrategy *bidstrategy.MockResourceBidStrategy
	mockExecutionStore   store.ExecutionStore
	mockCallback         *compute.MockCallback
	mockExecutor         *compute.MockExecutor
	bidder               compute.Bidder
}

func TestBidderSuite(t *testing.T) {
	suite.Run(t, new(BidderSuite))
}

func (s *BidderSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockSemanticStrategy = bidstrategy.NewMockSemanticBidStrategy(s.ctrl)
	s.mockResourceStrategy = bidstrategy.NewMockResourceBidStrategy(s.ctrl)
	s.mockExecutionStore = inmemory.NewStore()
	s.mockCallback = compute.NewMockCallback(s.ctrl)
	s.mockExecutor = compute.NewMockExecutor(s.ctrl)
	s.bidder = compute.NewBidder(compute.BidderParams{
		NodeID:           "testNodeID",
		SemanticStrategy: s.mockSemanticStrategy,
		ResourceStrategy: s.mockResourceStrategy,
		Store:            s.mockExecutionStore,
		Callback:         s.mockCallback,
		Executor:         s.mockExecutor,
		GetApproveURL: func() *url.URL {
			return &url.URL{}
		},
	})
}

func (s *BidderSuite) TestRunBidding_WithPendingApproval() {
	ctx := context.Background()
	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	askForBidRequest := compute.AskForBidRequest{
		Execution:       execution,
		WaitForApproval: true,
	}

	usageCalculator := capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{Defaults: models.Resources{}})

	tests := []struct {
		name                   string
		expectedExecutionState store.LocalStateType
		mockExpectations       func()
	}{
		{
			name: "semantic should not bid and should not wait; resource not evaluated.",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: false}, nil)
				s.mockCallback.EXPECT().OnBidComplete(ctx, NewBidResponseMatcher(false))
			},
		},
		{
			name:                   "semantic and resource should bid and not wait; bid complete",
			expectedExecutionState: store.ExecutionStateCreated,
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockCallback.EXPECT().OnBidComplete(ctx, NewBidResponseMatcher(true))
			},
		},
		{
			name:                   "semantic should wait resource should wait; bid NOT complete.",
			expectedExecutionState: store.ExecutionStateCreated,
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
			},
		},
		{
			name:                   "semantic should wait and resource should bid; bid NOT complete.",
			expectedExecutionState: store.ExecutionStateCreated,
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
			},
		},
		{
			name:                   "semantic should bid and resource should wait; bid NOT complete.",
			expectedExecutionState: store.ExecutionStateCreated,
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
			},
		},
		{
			name: "semantic bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("semantic error"))
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name: "resource bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("resource error"))
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			tt.mockExpectations()
			s.bidder.RunBidding(ctx, askForBidRequest, usageCalculator)

			exec, err := s.mockExecutionStore.GetExecution(ctx, askForBidRequest.Execution.ID)
			if tt.expectedExecutionState.IsUndefined() {
				s.Require().Error(err, "expected no execution to be created, but found one with state: %s", exec.State)
			} else {
				s.Equal(tt.expectedExecutionState, exec.State, "expected: %s, actual: %s", tt.expectedExecutionState, exec.State)
			}
		})
	}
}

func (s *BidderSuite) TestRunBidding_WithoutPendingApproval() {
	ctx := context.Background()
	askForBidRequest := compute.AskForBidRequest{
		Execution:       mock.ExecutionForJob(mock.Job()),
		WaitForApproval: false,
	}

	usageCalculator := capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{Defaults: models.Resources{}})

	tests := []struct {
		name                   string
		expectedExecutionState store.LocalStateType
		mockExpectations       func()
	}{
		{
			name: "semantic should not bid and should not wait; resource not evaluated.",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: false}, nil)
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name:                   "semantic and resource should bid and not wait; start running",
			expectedExecutionState: store.ExecutionStateBidAccepted,
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockExecutor.EXPECT().Run(ctx, gomock.Any())
			},
		},
		{
			name: "semantic should wait resource should wait; fail.",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name: "semantic should wait and resource should bid; fail.",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name: "semantic should bid and resource should wait; fail.",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name: "semantic bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("semantic error"))
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
		{
			name: "resource bid error",
			mockExpectations: func() {
				s.mockSemanticStrategy.EXPECT().ShouldBid(ctx, gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				s.mockResourceStrategy.EXPECT().ShouldBidBasedOnUsage(ctx, gomock.Any(), gomock.Any()).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("resource error"))
				s.mockCallback.EXPECT().OnComputeFailure(ctx, gomock.Any())
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.SetupTest()
			tt.mockExpectations()
			s.bidder.RunBidding(ctx, askForBidRequest, usageCalculator)

			exec, err := s.mockExecutionStore.GetExecution(ctx, askForBidRequest.Execution.ID)
			if tt.expectedExecutionState.IsUndefined() {
				s.Require().Error(err, "expected no execution to be created, but found one with state: %s", exec.State)
			} else {
				s.Equal(tt.expectedExecutionState, exec.State, "expected: %s, actual: %s", tt.expectedExecutionState, exec.State)
			}
		})
	}
}

type BidResponseMatcher struct {
	accepted bool
}

func NewBidResponseMatcher(accepted bool) *BidResponseMatcher {
	return &BidResponseMatcher{
		accepted: accepted,
	}
}

func (m *BidResponseMatcher) Matches(x interface{}) bool {
	req, ok := x.(compute.BidResult)
	if !ok {
		return false
	}

	return req.Accepted == m.accepted
}

func (m *BidResponseMatcher) String() string {
	return fmt.Sprintf("isBidAccepted=%v", m.accepted)
}
