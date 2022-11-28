package backend

import "context"

type ChainedCallbackParams struct {
	Callbacks []Callback
}

// ChainedCallback Callback that chains multiple callbacks and delegates the calls to them.
type ChainedCallback struct {
	callbacks []Callback
}

func NewChainedCallback(params ChainedCallbackParams) *ChainedCallback {
	return &ChainedCallback{
		callbacks: params.Callbacks,
	}
}

func (c ChainedCallback) OnRunSuccess(ctx context.Context, executionID string, result RunResult) {
	for _, callback := range c.callbacks {
		callback.OnRunSuccess(ctx, executionID, result)
	}
}

func (c ChainedCallback) OnRunFailure(ctx context.Context, executionID string, err error) {
	for _, callback := range c.callbacks {
		callback.OnRunFailure(ctx, executionID, err)
	}
}

func (c ChainedCallback) OnPublishSuccess(ctx context.Context, executionID string, result PublishResult) {
	for _, callback := range c.callbacks {
		callback.OnPublishSuccess(ctx, executionID, result)
	}
}

func (c ChainedCallback) OnPublishFailure(ctx context.Context, executionID string, err error) {
	for _, callback := range c.callbacks {
		callback.OnPublishFailure(ctx, executionID, err)
	}
}

func (c ChainedCallback) OnCancelSuccess(ctx context.Context, executionID string, result CancelResult) {
	for _, callback := range c.callbacks {
		callback.OnCancelSuccess(ctx, executionID, result)
	}
}

func (c ChainedCallback) OnCancelFailure(ctx context.Context, executionID string, err error) {
	for _, callback := range c.callbacks {
		callback.OnCancelFailure(ctx, executionID, err)
	}
}

// compile-time interface check
var _ Callback = &ChainedCallback{}
