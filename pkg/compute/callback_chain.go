package compute

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

func (c ChainedCallback) OnRunComplete(ctx context.Context, result RunResult) {
	for _, callback := range c.callbacks {
		callback.OnRunComplete(ctx, result)
	}
}

func (c ChainedCallback) OnPublishComplete(ctx context.Context, result PublishResult) {
	for _, callback := range c.callbacks {
		callback.OnPublishComplete(ctx, result)
	}
}

func (c ChainedCallback) OnCancelComplete(ctx context.Context, result CancelResult) {
	for _, callback := range c.callbacks {
		callback.OnCancelComplete(ctx, result)
	}
}

func (c ChainedCallback) OnComputeFailure(ctx context.Context, err ComputeError) {
	for _, callback := range c.callbacks {
		callback.OnComputeFailure(ctx, err)
	}
}

// compile-time interface check
var _ Callback = &ChainedCallback{}
