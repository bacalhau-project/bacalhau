package backoff

import (
	"context"
)

// Noop implements a backoff strategy that does NOT backoff
// regardless of the number of attempts.
type Noop struct {
}

func NewNoop() *Noop {
	return &Noop{}
}

func (b *Noop) Backoff(ctx context.Context, attempts int) {
}

// compile time check whether the Noop implements the Backoff interface.
var _ Backoff = (*Noop)(nil)
