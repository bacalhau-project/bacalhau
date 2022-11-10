package requesternode

import (
	"time"
)

// DefaultJobNegotiationTimeout default timeout value to wait for enough bids to be submitted
const DefaultJobNegotiationTimeout = 2 * time.Minute

// DefaultJobExecutionTimeout default timeout value for running, verifying and publishing job results.
const DefaultJobExecutionTimeout = 30 * time.Minute

// DefaultMinJobNegotiationTimeout requester node will replace any job negotiation timeout that is less than
// this value with DefaultJobNegotiationTimeout.
const DefaultMinJobNegotiationTimeout = 0 * time.Second

// DefaultMinJobExecutionTimeout requester node will replace any job execution timeout that is less than
// this value with DefaultJobExecutionTimeout.
const DefaultMinJobExecutionTimeout = 0 * time.Second

// DefaultStateManagerTaskInterval background task interval that periodically checks for expired states among other things.
const DefaultStateManagerTaskInterval = 30 * time.Second

type RequesterTimeoutConfig struct {
	// Timeout value waiting for enough bids to be submitted for a job
	JobNegotiationTimeout time.Duration

	// Timeout value for running, verifying and publishing job results, if the user didn't define one in the spec
	DefaultJobExecutionTimeout time.Duration

	// Requester node will replace any job negotiation timeout that is less than this
	// value with DefaultJobNegotiationTimeout.
	MinJobNegotiationTimeout time.Duration

	// Requester node will replace any job execution timeout that is less than this
	// value with DefaultJobExecutionTimeout.
	MinJobExecutionTimeout time.Duration
}

func NewDefaultRequesterTimeoutConfig() RequesterTimeoutConfig {
	return RequesterTimeoutConfig{
		JobNegotiationTimeout:      DefaultJobNegotiationTimeout,
		DefaultJobExecutionTimeout: DefaultJobExecutionTimeout,
		MinJobNegotiationTimeout:   DefaultMinJobNegotiationTimeout,
		MinJobExecutionTimeout:     DefaultMinJobExecutionTimeout,
	}
}

type RequesterNodeConfig struct {
	// configure the timeout for each shard state
	TimeoutConfig RequesterTimeoutConfig

	// background task interval that periodically checks for expired states among other things.
	StateManagerBackgroundTaskInterval time.Duration
}

func NewDefaultRequesterNodeConfig() RequesterNodeConfig {
	return RequesterNodeConfig{
		TimeoutConfig:                      NewDefaultRequesterTimeoutConfig(),
		StateManagerBackgroundTaskInterval: DefaultStateManagerTaskInterval,
	}
}

func populateDefaultConfigs(other RequesterNodeConfig) RequesterNodeConfig {
	config := other

	if config.TimeoutConfig.JobNegotiationTimeout == 0 {
		config.TimeoutConfig.JobNegotiationTimeout = DefaultJobNegotiationTimeout
	}
	if config.TimeoutConfig.DefaultJobExecutionTimeout == 0 {
		config.TimeoutConfig.DefaultJobExecutionTimeout = DefaultJobExecutionTimeout
	}
	if config.StateManagerBackgroundTaskInterval == 0 {
		config.StateManagerBackgroundTaskInterval = DefaultStateManagerTaskInterval
	}

	return config
}
