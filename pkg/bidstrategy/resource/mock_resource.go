package resource

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type MockResourceBidStrategy struct {
	mock.Mock
}

func (m *MockResourceBidStrategy) ShouldBidBasedOnUsage(ctx context.Context, request bidstrategy.BidStrategyRequest, usage model.ResourceUsageData) (bidstrategy.BidStrategyResponse, error) {
	args := m.Called(ctx, request, usage)
	return args.Get(0).(bidstrategy.BidStrategyResponse), args.Error(1)
}
