package compute

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
)

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

func (c ChainedCallback) OnBidComplete(ctx context.Context, result messages.BidResult) {
	for _, callback := range c.callbacks {
		callback.OnBidComplete(ctx, result)
	}
}

func (c ChainedCallback) OnRunComplete(ctx context.Context, result messages.RunResult) {
	for _, callback := range c.callbacks {
		callback.OnRunComplete(ctx, result)
	}
}

func (c ChainedCallback) OnCancelComplete(ctx context.Context, result messages.CancelResult) {
	for _, callback := range c.callbacks {
		callback.OnCancelComplete(ctx, result)
	}
}

func (c ChainedCallback) OnComputeFailure(ctx context.Context, err messages.ComputeError) {
	for _, callback := range c.callbacks {
		callback.OnComputeFailure(ctx, err)
	}
}

// compile-time interface check
var _ Callback = &ChainedCallback{}
