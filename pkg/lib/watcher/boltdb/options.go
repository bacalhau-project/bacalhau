package boltdb

import (
	"errors"
	"time"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

// EventStoreOption is a function type for configuring an EventStore.
// It allows for a flexible and extensible way to set options.
type EventStoreOption func(*eventStoreOptions)

// eventStoreOptions holds all configurable options for the EventStore.
type eventStoreOptions struct {
	eventsBucket       []byte             // Name of the bucket to store events
	checkpointBucket   []byte             // Name of the bucket to store checkpoints
	serializer         watcher.Serializer // Serializer used for event marshaling/unmarshaling
	cacheSize          int                // Size of the LRU cache for events
	longPollingTimeout time.Duration      // Timeout for long-polling requests
	gcAgeThreshold     time.Duration      // Age threshold for garbage collection
	gcCadence          time.Duration      // Frequency of garbage collection runs
	gcMaxRecordsPerRun int                // Maximum number of records to process per GC run
	gcMaxDuration      time.Duration      // Maximum duration for a single GC run
	clock              clock.Clock
}

// validate checks all options for validity.
// It returns an error if any option is invalid.
func (s *eventStoreOptions) validate() error {
	if s == nil {
		return errors.New("options cannot be nil")
	}
	return errors.Join(
		validate.NotBlank(string(s.eventsBucket), "eventsBucket cannot be blank"),
		validate.NotBlank(string(s.checkpointBucket), "checkpointBucket cannot be blank"),
		validate.NotNil(s.serializer, "serializer cannot be nil"),
		validate.IsGreaterOrEqualToZero(s.cacheSize, "cacheSize cannot be negative"),
		validate.IsGreaterThanZero(s.longPollingTimeout, "longPollingTimeout must be greater than zero"),
		validate.IsGreaterOrEqualToZero(s.gcAgeThreshold, "gcAgeThreshold cannot be negative"),
		validate.IsGreaterOrEqualToZero(s.gcCadence, "gcCadence cannot be negative"),
		validate.IsGreaterOrEqualToZero(s.gcMaxRecordsPerRun, "gcMaxRecordsPerRun cannot be negative"),
		validate.IsGreaterOrEqualToZero(s.gcMaxDuration, "gcMaxDuration cannot be negative"),
	)
}

// defaultEventStoreOptions returns the default options for an EventStore.
// These defaults can be overridden using the With* functions.
func defaultEventStoreOptions() *eventStoreOptions {
	return &eventStoreOptions{
		eventsBucket:       []byte("events"),
		checkpointBucket:   []byte("checkpoints"),
		serializer:         watcher.NewJSONSerializer(),
		cacheSize:          1000,
		longPollingTimeout: 10 * time.Second,
		gcAgeThreshold:     24 * time.Hour,
		gcCadence:          10 * time.Minute,
		gcMaxRecordsPerRun: 1000,
		gcMaxDuration:      10 * time.Second,
		clock:              clock.New(),
	}
}

// WithEventsBucket sets the name of the bucket used to store events.
func WithEventsBucket(name string) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.eventsBucket = []byte(name)
	}
}

// WithCheckpointBucket sets the name of the bucket used to store checkpoints.
func WithCheckpointBucket(name string) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.checkpointBucket = []byte(name)
	}
}

// WithEventSerializer sets the serializer used for events.
// This allows for custom serialization formats if needed.
func WithEventSerializer(serializer watcher.Serializer) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.serializer = serializer
	}
}

// WithCacheSize sets the size of the LRU cache used to store events.
// A larger cache can improve performance but uses more memory.
func WithCacheSize(size int) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.cacheSize = size
	}
}

// WithLongPollingTimeout sets the timeout duration for long-polling requests.
// This determines how long a client will wait for new events before the request times out.
func WithLongPollingTimeout(timeout time.Duration) EventStoreOption {
	return func(o *eventStoreOptions) {
		o.longPollingTimeout = timeout
	}
}

// WithGCAgeThreshold sets the age threshold for event pruning.
// Events older than this will be considered for pruning during garbage collection.
func WithGCAgeThreshold(threshold time.Duration) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.gcAgeThreshold = threshold
	}
}

// WithGCCadence sets the interval at which garbage collection runs.
// More frequent GC can keep the database size down but may impact performance.
func WithGCCadence(cadence time.Duration) EventStoreOption {
	return func(s *eventStoreOptions) {
		s.gcCadence = cadence
	}
}

// WithGCMaxRecordsPerRun sets the maximum number of records to process in a single GC run.
// This helps limit the duration of GC operations to avoid long-running transactions.
func WithGCMaxRecordsPerRun(max int) EventStoreOption {
	return func(o *eventStoreOptions) {
		o.gcMaxRecordsPerRun = max
	}
}

// WithGCMaxDuration sets the maximum duration for a single GC run.
// This provides another way to limit GC operations to avoid long-running transactions.
func WithGCMaxDuration(duration time.Duration) EventStoreOption {
	return func(o *eventStoreOptions) {
		o.gcMaxDuration = duration
	}
}

// WithClock sets the clock used for time-based operations.
// This is useful for testing to provide a mockable clock.
func WithClock(clock clock.Clock) EventStoreOption {
	return func(o *eventStoreOptions) {
		o.clock = clock
	}
}
