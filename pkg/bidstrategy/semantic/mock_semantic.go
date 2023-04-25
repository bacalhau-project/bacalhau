package semantic

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
)

type MockSemanticBidStrategy struct {
	mock.Mock
}

func (m *MockSemanticBidStrategy) ShouldBid(ctx context.Context, request bidstrategy.BidStrategyRequest) (bidstrategy.BidStrategyResponse, error) {
	args := m.Called(ctx, request)
	return args.Get(0).(bidstrategy.BidStrategyResponse), args.Error(1)
}
