package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan struct{}, 1)
	seenHandler := false
	OnCancel(ctx, func() {
		seenHandler = true
		ch <- struct{}{}
	})

	cancel()
	<-ch
	require.True(t, seenHandler, "OnCancel() callback not called")
}
