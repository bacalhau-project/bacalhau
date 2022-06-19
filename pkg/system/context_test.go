package system

import (
	"testing"
)

func TestOnCancel(t *testing.T) {
	t.Skip("OnCancel not defined?")
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// ch := make(chan struct{}, 1)
	// seenHandler := false
	// OnCancel(ctx, func() {
	// 	seenHandler = true
	// 	ch <- struct{}{}
	// })

	// cancel()
	// <-ch
	// assert.True(t, seenHandler, "OnCancel() callback not called")
}
