//go:generate mockgen --source types.go --destination mock_backoff.go --package backoff
package backoff

import (
	"context"
	"time"
)

// Backoff is the interface for backoff.
type Backoff interface {
	// Backoff waits and blocks the caller for a duration of time depending on the number of attempts,
	// or until the context is done.
	Backoff(ctx context.Context, attempts int)

	// BackoffDuration returns the duration that would be used for the given number of attempts
	// without actually performing the wait.
	BackoffDuration(attempts int) time.Duration
}
