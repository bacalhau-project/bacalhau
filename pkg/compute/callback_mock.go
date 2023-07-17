package compute

import "context"

type CallbackMock struct {
	OnBidCompleteHandler    func(ctx context.Context, result BidResult)
	OnCancelCompleteHandler func(ctx context.Context, result CancelResult)
	OnComputeFailureHandler func(ctx context.Context, err ComputeError)
	OnRunCompleteHandler    func(ctx context.Context, result RunResult)
}

// OnBidComplete implements Callback
func (c CallbackMock) OnBidComplete(ctx context.Context, result BidResult) {
	if c.OnBidCompleteHandler != nil {
		c.OnBidCompleteHandler(ctx, result)
	}
}

// OnCancelComplete implements Callback
func (c CallbackMock) OnCancelComplete(ctx context.Context, result CancelResult) {
	if c.OnCancelCompleteHandler != nil {
		c.OnCancelCompleteHandler(ctx, result)
	}
}

// OnComputeFailure implements Callback
func (c CallbackMock) OnComputeFailure(ctx context.Context, err ComputeError) {
	if c.OnComputeFailureHandler != nil {
		c.OnComputeFailureHandler(ctx, err)
	}
}

// OnRunComplete implements Callback
func (c CallbackMock) OnRunComplete(ctx context.Context, result RunResult) {
	if c.OnRunCompleteHandler != nil {
		c.OnRunCompleteHandler(ctx, result)
	}
}

var _ Callback = CallbackMock{}
