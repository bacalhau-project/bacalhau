package compute

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type MockCallbackTestify struct {
	// TODO: migrate tests to use the generated gomock instead of testify
	mock.Mock
}

func (m *MockCallbackTestify) OnRunComplete(ctx context.Context, result RunResult) {
	m.Called(ctx, result)
}

func (m *MockCallbackTestify) OnCancelComplete(ctx context.Context, result CancelResult) {
	m.Called(ctx, result)
}

func (m *MockCallbackTestify) OnComputeFailure(ctx context.Context, failure ComputeError) {
	m.Called(ctx, failure)
}

func (m *MockCallbackTestify) OnBidComplete(ctx context.Context, result BidResult) {
	m.Called(ctx, result)
}
