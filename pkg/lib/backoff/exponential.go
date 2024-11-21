package backoff

import (
	"context"
	"math"
	"time"
)

// Exponential implements a backoff strategy that increases the backoff duration exponentially,
// up to a maximum backoff duration.
type Exponential struct {
	BaseBackoff time.Duration // Base backoff duration
	MaxBackoff  time.Duration // Maximum backoff duration
}

func NewExponential(baseBackoff, maxBackoff time.Duration) *Exponential {
	return &Exponential{
		BaseBackoff: baseBackoff,
		MaxBackoff:  maxBackoff,
	}
}

func (eb *Exponential) Backoff(ctx context.Context, attempts int) {
	if attempts == 0 {
		return
	}

	backoffDuration := eb.BackoffDuration(attempts)
	select {
	case <-time.After(backoffDuration):
	case <-ctx.Done():
	}
}

func (eb *Exponential) BackoffDuration(attempts int) time.Duration {
	if attempts == 0 {
		return 0
	}

	backoff := float64(eb.BaseBackoff) * math.Pow(2, float64(attempts-1))
	if backoff > float64(eb.MaxBackoff) {
		backoff = float64(eb.MaxBackoff)
	}
	return time.Duration(backoff)
}

// compile time check whether the Exponential implements the Backoff interface.
var _ Backoff = (*Exponential)(nil)
