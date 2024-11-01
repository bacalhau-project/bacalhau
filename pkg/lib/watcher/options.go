package watcher

import (
	"errors"
	mathgo "math"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

type RetryStrategy int

const (
	RetryStrategyBlock RetryStrategy = iota
	RetryStrategySkip
)

// WatchOption is a function type for configuring watch options
type WatchOption func(*watchOptions)

// watchOptions holds configuration options for watching events
type watchOptions struct {
	initialEventIterator EventIterator // starting position for watching if no checkpoint is found
	filter               EventFilter   // filter for events
	batchSize            int           // number of events to fetch in each batch
	initialBackoff       time.Duration // initial backoff duration for retries
	maxBackoff           time.Duration // maximum backoff duration for retries
	maxRetries           int
	retryStrategy        RetryStrategy
}

// validate checks all options for validity
func (o *watchOptions) validate() error {
	return errors.Join(
		validate.IsGreaterThanZero(o.batchSize, "batchSize must be greater than zero"),
		validate.IsGreaterOrEqualToZero(o.initialBackoff, "initialBackoff cannot be negative"),
		validate.IsGreaterOrEqualToZero(o.maxBackoff, "maxBackoff cannot be negative"),
		validate.IsGreaterOrEqualToZero(o.maxRetries, "maxRetries cannot be negative"),
		validate.IsGreaterOrEqual(o.maxBackoff, o.initialBackoff, "maxBackoff must be greater than or equal to initialBackoff"))
}

// defaultWatchOptions returns the default watch options
func defaultWatchOptions() *watchOptions {
	return &watchOptions{
		initialEventIterator: TrimHorizonIterator(),
		batchSize:            100,
		initialBackoff:       200 * time.Millisecond,
		maxBackoff:           3 * time.Minute,
		maxRetries:           mathgo.MaxInt, // infinite retries
		retryStrategy:        RetryStrategyBlock,
	}
}

// WithInitialEventIterator sets the starting position for watching if no checkpoint is found
func WithInitialEventIterator(iterator EventIterator) WatchOption {
	return func(o *watchOptions) {
		o.initialEventIterator = iterator
	}
}

// WithFilter sets the event filter for watching
func WithFilter(filter EventFilter) WatchOption {
	return func(o *watchOptions) {
		o.filter = filter
	}
}

// WithBatchSize sets the number of events to fetch in each batch
func WithBatchSize(size int) WatchOption {
	return func(o *watchOptions) {
		o.batchSize = size
	}
}

// WithInitialBackoff sets the initial backoff duration for retries
func WithInitialBackoff(backoff time.Duration) WatchOption {
	return func(o *watchOptions) {
		o.initialBackoff = backoff
	}
}

// WithMaxBackoff sets the maximum backoff duration for retries
func WithMaxBackoff(backoff time.Duration) WatchOption {
	return func(o *watchOptions) {
		o.maxBackoff = backoff
	}
}

// WithMaxRetries sets the maximum number of retries for event handling
func WithMaxRetries(maxRetries int) WatchOption {
	return func(o *watchOptions) {
		o.maxRetries = maxRetries
	}
}

// WithRetryStrategy sets the retry strategy for event handling
func WithRetryStrategy(strategy RetryStrategy) WatchOption {
	return func(o *watchOptions) {
		o.retryStrategy = strategy
	}
}
