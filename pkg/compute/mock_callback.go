package compute

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockCallback struct {
	mock.Mock
}

func (m *MockCallback) OnRunComplete(ctx context.Context, result RunResult) {
	m.Called(ctx, result)
}

func (m *MockCallback) OnPublishComplete(ctx context.Context, result PublishResult) {
	m.Called(ctx, result)
}

func (m *MockCallback) OnCancelComplete(ctx context.Context, result CancelResult) {
	m.Called(ctx, result)
}

func (m *MockCallback) OnComputeFailure(ctx context.Context, failure ComputeError) {
	m.Called(ctx, failure)
}

func (m *MockCallback) OnBidComplete(ctx context.Context, result BidResult) {
	m.Called(ctx, result)
}
