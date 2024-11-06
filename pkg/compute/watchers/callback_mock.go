package watchers

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages/legacy"
)

type CallbackMock struct {
	OnBidCompleteHandler    func(ctx context.Context, result legacy.BidResult)
	OnComputeFailureHandler func(ctx context.Context, err legacy.ComputeError)
	OnRunCompleteHandler    func(ctx context.Context, result legacy.RunResult)
}

// OnBidComplete implements Callback
func (c CallbackMock) OnBidComplete(ctx context.Context, result legacy.BidResult) {
	if c.OnBidCompleteHandler != nil {
		c.OnBidCompleteHandler(ctx, result)
	}
}

// OnComputeFailure implements Callback
func (c CallbackMock) OnComputeFailure(ctx context.Context, err legacy.ComputeError) {
	if c.OnComputeFailureHandler != nil {
		c.OnComputeFailureHandler(ctx, err)
	}
}

// OnRunComplete implements Callback
func (c CallbackMock) OnRunComplete(ctx context.Context, result legacy.RunResult) {
	if c.OnRunCompleteHandler != nil {
		c.OnRunCompleteHandler(ctx, result)
	}
}

var _ compute.Callback = CallbackMock{}
