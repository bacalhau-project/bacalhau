package requester

import "context"

type RetryStrategy interface {
	// ShouldRetry returns true if the job can be retried.
	ShouldRetry(ctx context.Context, request RetryRequest) bool
}

type RetryRequest struct {
	JobID string
}
