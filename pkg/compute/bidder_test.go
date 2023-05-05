//go:build unit || !integration

package compute_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/resource"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/mockstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var (
	mockSemanticStrategy *semantic.MockSemanticBidStrategy
	mockResourceStrategy *resource.MockResourceBidStrategy
	mockExecutionStore   *mockstore.MockExecutionStore
	mockCallback         *compute.MockCallback
	bidder               compute.Bidder
)

func TestRunBidding(t *testing.T) {
	ctx := context.Background()
	job, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(t, err)
	askForBidRequest := compute.AskForBidRequest{
		Job: *job,
	}

	usageCalculator := capacity.NewDefaultsUsageCalculator(capacity.DefaultsUsageCalculatorParams{Defaults: model.ResourceUsageData{}})

	resetMocks := func() {
		mockSemanticStrategy = new(semantic.MockSemanticBidStrategy)
		mockResourceStrategy = new(resource.MockResourceBidStrategy)
		mockExecutionStore = new(mockstore.MockExecutionStore)
		mockCallback = new(compute.MockCallback)
		bidder = compute.NewBidder(compute.BidderParams{
			NodeID:           "testNodeID",
			SemanticStrategy: mockSemanticStrategy,
			ResourceStrategy: mockResourceStrategy,
			Store:            mockExecutionStore,
			Callback:         mockCallback,
			GetApproveURL: func() *url.URL {
				return &url.URL{}
			},
		})
	}

	tests := []struct {
		name                string
		semanticBidResponse bidstrategy.BidStrategyResponse
		resourceBidResponse bidstrategy.BidStrategyResponse
		mockExpectations    func()
	}{
		{
			name:                "semantic should not bid and should not wait; resource not evaluated.",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: false},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: false}, nil)
				mockCallback.On("OnBidComplete", ctx, mock.Anything).
					Return()
			},
		},
		{
			name:                "semantic and resource should bid and not wait; bid complete",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false},
			resourceBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				mockResourceStrategy.On("ShouldBidBasedOnUsage", ctx, mock.Anything, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				mockExecutionStore.On("CreateExecution", ctx, mock.Anything).
					Return(nil)
				mockCallback.On("OnBidComplete", ctx, mock.Anything).
					Return()
			},
		},
		{
			name:                "semantic should wait resource should wait; bid NOT complete.",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true},
			resourceBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				mockResourceStrategy.On("ShouldBidBasedOnUsage", ctx, mock.Anything, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				mockExecutionStore.On("CreateExecution", ctx, mock.Anything).
					Return(nil)
			},
		},
		{
			name:                "semantic should wait and resource should bid; bid NOT complete.",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true},
			resourceBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				mockResourceStrategy.On("ShouldBidBasedOnUsage", ctx, mock.Anything, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				mockExecutionStore.On("CreateExecution", ctx, mock.Anything).
					Return(nil)
			},
		},
		{
			name:                "semantic should bid and resource should wait; bid NOT complete.",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false},
			resourceBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				mockResourceStrategy.On("ShouldBidBasedOnUsage", ctx, mock.Anything, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: false, ShouldWait: true}, nil)
				mockExecutionStore.On("CreateExecution", ctx, mock.Anything).
					Return(nil)
			},
		},
		{
			name: "semantic bid error",
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("semantic error"))
				mockCallback.On("OnComputeFailure", ctx, mock.Anything).
					Return()
			},
		},
		{
			name:                "resource bid error",
			semanticBidResponse: bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false},
			mockExpectations: func() {
				mockSemanticStrategy.On("ShouldBid", ctx, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{ShouldBid: true, ShouldWait: false}, nil)
				mockResourceStrategy.On("ShouldBidBasedOnUsage", ctx, mock.Anything, mock.Anything).
					Return(bidstrategy.BidStrategyResponse{}, errors.New("resource error"))
				mockCallback.On("OnComputeFailure", ctx, mock.Anything).
					Return()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetMocks()
			tt.mockExpectations()

			bidder.RunBidding(ctx, askForBidRequest, usageCalculator)

			mockSemanticStrategy.AssertExpectations(t)
			mockResourceStrategy.AssertExpectations(t)
			mockExecutionStore.AssertExpectations(t)
			mockCallback.AssertExpectations(t)
		})
	}
}
