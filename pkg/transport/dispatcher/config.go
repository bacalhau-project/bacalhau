package dispatcher

import (
	"errors"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

const (
	// Default checkpoint settings
	defaultCheckpointInterval = 5 * time.Second
	defaultCheckpointTimeout  = 5 * time.Second

	// Default stall detection settings
	defaultStallTimeout       = 5 * time.Minute
	defaultStallCheckInterval = 30 * time.Second

	// Default processing settings
	defaultProcessInterval = 10 * time.Millisecond
	defaultSeekTimeout     = 30 * time.Second

	// Default retry settings
	defaultBaseRetryInterval = 5 * time.Second
	defaultMaxRetryInterval  = 5 * time.Minute
)

// Config defines the configuration settings for the dispatcher. It controls various
// timeouts, intervals and retry behavior for the event dispatch process.
type Config struct {
	// CheckpointInterval determines how often the dispatcher saves its progress.
	// Lower values provide better durability at the cost of more IO operations.
	// Negative values disable checkpointing.
	// Default: 5 seconds
	CheckpointInterval time.Duration

	// CheckpointTimeout is the maximum time allowed for a checkpoint operation.
	// If exceeded, the checkpoint is abandoned and will be retried next interval.
	// Default: 5 seconds
	CheckpointTimeout time.Duration

	// StallTimeout is the duration after which a pending message is considered stalled.
	// When a message is stalled, recovery mechanisms may be triggered.
	// Should be significantly longer than expected network latency and processing time.
	// Default: 5 minutes
	StallTimeout time.Duration

	// StallCheckInterval defines how often to check for stalled messages.
	// More frequent checks provide quicker detection but increase CPU usage.
	// Should be shorter than StallTimeout.
	// Default: 30 seconds
	StallCheckInterval time.Duration

	// ProcessInterval controls how frequently the dispatcher processes publish results.
	// Lower values reduce latency but increase CPU usage.
	// Default: 10 milliseconds
	ProcessInterval time.Duration

	// SeekTimeout is the maximum time allowed for seeking to a position in the event stream.
	// Exceeding this timeout indicates potential issues with the event source.
	// Default: 30 seconds
	SeekTimeout time.Duration

	// BaseRetryInterval is the initial delay between retry attempts after a failure.
	// This interval increases exponentially up to MaxRetryInterval.
	// Default: 5 seconds
	BaseRetryInterval time.Duration

	// MaxRetryInterval caps the maximum delay between retry attempts.
	// Prevents exponential backoff from growing too large.
	// Default: 5 minutes
	MaxRetryInterval time.Duration
}

// DefaultConfig returns a Config initialized with reasonable default values.
// These defaults are designed to work well for most use cases but can be
// overridden based on specific requirements.
func DefaultConfig() Config {
	return Config{
		CheckpointInterval: defaultCheckpointInterval,
		CheckpointTimeout:  defaultCheckpointTimeout,
		StallTimeout:       defaultStallTimeout,
		StallCheckInterval: defaultStallCheckInterval,
		ProcessInterval:    defaultProcessInterval,
		SeekTimeout:        defaultSeekTimeout,
		BaseRetryInterval:  defaultBaseRetryInterval,
		MaxRetryInterval:   defaultMaxRetryInterval,
	}
}

// Validate checks if the configuration values are valid and returns an error if not.
// This helps catch configuration issues early before they cause problems at runtime.
func (c *Config) Validate() error {
	return errors.Join(
		// Intervals must be positive
		validate.IsGreaterThanZero(c.ProcessInterval, "ProcessInterval must be positive"),
		validate.IsGreaterThanZero(c.StallCheckInterval, "StallCheckInterval must be positive"),

		// Timeouts must be positive
		validate.IsGreaterThanZero(c.CheckpointTimeout, "CheckpointTimeout must be positive"),
		validate.IsGreaterThanZero(c.StallTimeout, "StallTimeout must be positive"),
		validate.IsGreaterThanZero(c.SeekTimeout, "SeekTimeout must be positive"),

		// Retry intervals must be positive and properly ordered
		validate.IsGreaterThanZero(c.BaseRetryInterval, "BaseRetryInterval must be positive"),
		validate.IsGreaterThanZero(c.MaxRetryInterval, "MaxRetryInterval must be positive"),
		validate.True(
			c.MaxRetryInterval >= c.BaseRetryInterval,
			"MaxRetryInterval must be greater than or equal to BaseRetryInterval",
		),

		// Logical relationships between intervals
		validate.True(
			c.StallCheckInterval < c.StallTimeout,
			"StallCheckInterval must be less than StallTimeout",
		),
	)
}
