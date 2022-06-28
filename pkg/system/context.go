package system

import (
	"context"
	"os"
	"os/signal"
)

// WithSignalShutdown returns a copy of the parent context which cancels
// itself if a "ctrl+c" interrupt signal is captured. The returned cancel
// function cleans up the resources associated with this context and should
// be called as soon as the operations in this context complete.
func WithSignalShutdown(parent context.Context) (context.Context, context.CancelFunc) {
	ch := make(chan os.Signal, 1)
	signal.Reset(os.Interrupt)
	signal.Notify(ch, os.Interrupt)

	ctx, cancel := context.WithCancel(parent)
	go func(ch chan os.Signal, cancel context.CancelFunc) {
		select {
		case <-ch:
			cancel()

		// Clean-up goroutine if the context is canceled:
		case <-ctx.Done():
		}
	}(ch, cancel)

	return ctx, cancel
}

// OnCancel calls the given callback function when the provided context is
// canceled. Can be used to register clean-up callbacks for long-running
// system contexts.
func OnCancel(ctx context.Context, fn func()) {
	go func(ch <-chan struct{}, fn func()) {
		<-ch
		fn()
	}(ctx.Done(), fn)
}
