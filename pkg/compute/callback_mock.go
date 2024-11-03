package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

type CallbackMock struct {
	OnBidCompleteHandler    func(ctx context.Context, result messages.BidResult)
	OnCancelCompleteHandler func(ctx context.Context, result messages.CancelResult)
	OnComputeFailureHandler func(ctx context.Context, err messages.ComputeError)
	OnRunCompleteHandler    func(ctx context.Context, result messages.RunResult)
}

// OnBidComplete implements Callback
func (c CallbackMock) OnBidComplete(ctx context.Context, result messages.BidResult) {
	if c.OnBidCompleteHandler != nil {
		c.OnBidCompleteHandler(ctx, result)
	}
}

// OnComputeFailure implements Callback
func (c CallbackMock) OnComputeFailure(ctx context.Context, err messages.ComputeError) {
	if c.OnComputeFailureHandler != nil {
		c.OnComputeFailureHandler(ctx, err)
	}
}

// OnRunComplete implements Callback
func (c CallbackMock) OnRunComplete(ctx context.Context, result messages.RunResult) {
	if c.OnRunCompleteHandler != nil {
		c.OnRunCompleteHandler(ctx, result)
	}
}

var _ Callback = CallbackMock{}
