//go:build unit || !integration

package util

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type mockBidStrategy func(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error)

// ShouldBid implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	return (*m)(ctx, request)
}

// ShouldBidBasedOnUsage implements bidstrategy.BidStrategy
func (m *mockBidStrategy) ShouldBidBasedOnUsage(ctx context.Context, request bidstrategy.BidStrategyRequest, resourceUsage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	return (*m)(ctx, request)
}

var (
	returnTrue = mockBidStrategy(func(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
		return bidstrategy.BidStrategyResponse{ShouldBid: true}, nil
	})
	returnFalse = mockBidStrategy(func(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
		return bidstrategy.BidStrategyResponse{ShouldBid: false}, nil
	})
	returnExplosion = mockBidStrategy(func(context.Context, bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
		return bidstrategy.BidStrategyResponse{ShouldBid: false}, fmt.Errorf("should not happen")
	})
)

var _ bidstrategy.BidStrategy = (*mockBidStrategy)(nil)

func TestExecutorsBidStrategy(t *testing.T) {
	for _, testCase := range []struct {
		name             string
		installed        bool
		executorStrategy bidstrategy.BidStrategy
		check            func(require.TestingT, bool, ...any)
	}{
		{"executor not installed", false, &returnExplosion, require.False},
		{"executor installed/strategy says no", true, &returnFalse, require.False},
		{"executor installed/strategy says yes", true, &returnTrue, require.True},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			noop_provider := NewNoopExecutors(noop.ExecutorConfig{
				ExternalHooks: noop.ExecutorConfigExternalHooks{
					IsInstalled: func(ctx context.Context) (bool, error) {
						return testCase.installed, nil
					},
					GetBidStrategy: func(ctx context.Context) (bidstrategy.BidStrategy, error) {
						return testCase.executorStrategy, nil
					},
				},
			})
			strategy := NewExecutorSpecificBidStrategy(noop_provider)
			result, err := strategy.ShouldBid(context.Background(), bidstrategy.BidStrategyRequest{
				Job: model.Job{
					Spec: model.Spec{Engine: model.EngineNoop},
				},
			})
			require.NoError(t, err)
			testCase.check(t, result.ShouldBid, fmt.Sprintf("Reason: %q", result.Reason))
		})
	}
}
